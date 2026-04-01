package controller

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"io"
	"time"

	hkdf "golang.org/x/crypto/hkdf"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	secretderiverv1alpha1 "github.com/LightJack05/SecretDeriver/api/v1alpha1"
)

// DerivedSecretReconciler reconciles a DerivedSecret object
type DerivedSecretReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=secretderiver.lightjack.de,resources=derivedsecrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=secretderiver.lightjack.de,resources=derivedsecrets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=secretderiver.lightjack.de,resources=derivedsecrets/finalizers,verbs=update

// Allow reading secrets globally
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// Allow writing secrets in the same namespace as the DerivedSecret
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the DerivedSecret object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/reconcile
func (r *DerivedSecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)
	log.Info("Got request to reconcile object", "namespace", req.Namespace, "name", req.Name)

	derivedSecret := &secretderiverv1alpha1.DerivedSecret{}
	if err := r.Get(ctx, req.NamespacedName, derivedSecret); err != nil {
		log.Error(err, "unable to fetch DerivedSecret")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	parentSecretRef := derivedSecret.Spec.ParentSecretRef
	parentSecretKey := derivedSecret.Spec.ParentSecretKey

	parentSecret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: parentSecretRef.Namespace, Name: parentSecretRef.Name}, parentSecret); err != nil {
		log.Error(err, "unable to fetch parent secret", "namespace", parentSecretRef.Namespace, "name", parentSecretRef.Name)
		if err := r.handleParentSecretNotFound(ctx, derivedSecret); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	sourceValue, exists := parentSecret.Data[parentSecretKey]
	if !exists {
		log.Info("Specified key not found in parent secret", "key", parentSecretKey)
		if err := r.handleKeyNotFoundInParent(ctx, derivedSecret); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	if len(sourceValue) == 0 {
		log.Info("Specified key in parent secret has an empty value", "key", parentSecretKey)
		if err := r.handleEmptyValueInParent(ctx, derivedSecret); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
	}

	derivedValue, err := r.deriveValue(ctx, derivedSecret, sourceValue)
	if err != nil {
		log.Error(err, "failed to derive value")
		if err := r.handleDerivationError(ctx, derivedSecret, err); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, err
	}

	if err := r.createDerivedSecret(ctx, derivedSecret, derivedValue); err != nil {
		log.Error(err, "failed to create or update derived secret")
		return ctrl.Result{}, err
	}

	if err := r.updateStatusReady(ctx, derivedSecret); err != nil {
		log.Error(err, "failed to update DerivedSecret status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *DerivedSecretReconciler) createDerivedSecret(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret, derivedValue []byte) error {
	secret := &corev1.Secret{}
	if err := r.Get(ctx, client.ObjectKey{Namespace: derivedSecret.Namespace, Name: derivedSecret.Name}, secret); err != nil {
		if client.IgnoreNotFound(err) != nil {
			return fmt.Errorf("failed to check for existing derived secret: %w", err)
		}

		// Secret does not exist, create it
		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      derivedSecret.Name,
				Namespace: derivedSecret.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(derivedSecret, secretderiverv1alpha1.GroupVersion.WithKind("DerivedSecret")),
				},
			},
			Data: map[string][]byte{
				derivedSecret.Spec.GeneratedSecretKey: derivedValue,
			},
		}
		if err := r.Create(ctx, newSecret); err != nil {
			return fmt.Errorf("failed to create derived secret: %w", err)
		}
	} else {
		newSecretContent := map[string][]byte{
			derivedSecret.Spec.GeneratedSecretKey: derivedValue,
		}
		if !equalSecretData(secret.Data, newSecretContent) {
			secret.Data = newSecretContent
			if err := r.Update(ctx, secret); err != nil {
				return fmt.Errorf("failed to update derived secret: %w", err)
			}
		}
	}

	return nil
}

func equalSecretData(presentSecretContent map[string][]byte, newSecretContent map[string][]byte) bool {
	if len(presentSecretContent) != len(newSecretContent) {
		return false
	}
	for key, newValue := range newSecretContent {
		presentValue, exists := presentSecretContent[key]
		if !exists || subtle.ConstantTimeCompare(presentValue, newValue) != 1 {
			return false
		}
	}
	return true
}

func (r *DerivedSecretReconciler) deriveValue(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret, sourceValue []byte) ([]byte, error) {
	salt := []byte(derivedSecret.Namespace + "/" + derivedSecret.Name)
	hkdfReader := hkdf.New(sha256.New, sourceValue, salt, nil)

	derivedValue := make([]byte, 32)
	_, err := io.ReadFull(hkdfReader, derivedValue)
	if err != nil {
		return nil, fmt.Errorf("failed to derive value using HKDF: %w", err)
	}

	return derivedValue, nil
}

func (r *DerivedSecretReconciler) updateStatusReady(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret) error {
	derivedSecret.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionTrue,
			Reason:             "DerivationSuccessful",
			Message:            "The derived secret has been successfully created or updated.",
			LastTransitionTime: metav1.Now(),
		},
	}
	derivedSecret.Status.Phase = "Ready"
	if err := r.Status().Update(ctx, derivedSecret); err != nil {
		return fmt.Errorf("failed to update DerivedSecret status: %w", err)
	}
	return nil
}

func (r *DerivedSecretReconciler) handleDerivationError(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret, err error) error {
	derivedSecret.Status.Conditions = []metav1.Condition{
		{
			Type:               "Error",
			Status:             metav1.ConditionTrue,
			Reason:             "DerivationFailed",
			Message:            fmt.Sprintf("Failed to derive value: %v", err),
			LastTransitionTime: metav1.Now(),
		},
	}
	derivedSecret.Status.Phase = "Error"
	if err := r.Status().Update(ctx, derivedSecret); err != nil {
		return fmt.Errorf("failed to update DerivedSecret status: %w", err)
	}
	return nil
}

func (r *DerivedSecretReconciler) handleEmptyValueInParent(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret) error {
	derivedSecret.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "EmptyValue",
			Message:            "The specified key in the parent secret has an empty value.",
			LastTransitionTime: metav1.Now(),
		},
	}
	derivedSecret.Status.Phase = "Error"
	if err := r.Status().Update(ctx, derivedSecret); err != nil {
		return err
	}
	return nil
}

func (r *DerivedSecretReconciler) handleKeyNotFoundInParent(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret) error {
	derivedSecret.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "KeyNotFound",
			Message:            "The specified key does not exist in the parent secret.",
			LastTransitionTime: metav1.Now(),
		},
	}
	derivedSecret.Status.Phase = "Error"
	if err := r.Status().Update(ctx, derivedSecret); err != nil {
		return err
	}
	return nil
}

func (r *DerivedSecretReconciler) handleParentSecretNotFound(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret) error {
	derivedSecret.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "ParentSecretNotFound",
			Message:            "The specified parent secret does not exist.",
			LastTransitionTime: metav1.Now(),
		},
	}
	derivedSecret.Status.Phase = "Error"
	if err := r.Status().Update(ctx, derivedSecret); err != nil {
		return err
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DerivedSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretderiverv1alpha1.DerivedSecret{}).
		Named("derivedsecret").
		Owns(&corev1.Secret{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findDerivedSecretsForParent),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *DerivedSecretReconciler) findDerivedSecretsForParent(ctx context.Context, secret client.Object) []reconcile.Request {
	derivedSecretList := &secretderiverv1alpha1.DerivedSecretList{}
	if err := r.List(ctx, derivedSecretList); err != nil {
		return nil
	}

	var requests []reconcile.Request
	for _, ds := range derivedSecretList.Items {
		if ds.Spec.ParentSecretRef.Name == secret.GetName() && ds.Spec.ParentSecretRef.Namespace == secret.GetNamespace() {
			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      ds.Name,
					Namespace: ds.Namespace,
				},
			})
		}
	}
	return requests
}
