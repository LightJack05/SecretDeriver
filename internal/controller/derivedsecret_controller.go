package controller

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
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

// DerivedSecretReconciler reconciles DerivedSecret resources.
// For each DerivedSecret it reads a value from the referenced parent Secret, derives a new
// value using HKDF-SHA256 (salted with the resource's namespace/name), and writes the result
// into a managed Kubernetes Secret of the same name and namespace.
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

// Reconcile fetches the DerivedSecret, resolves the parent secret, derives the value, and
// ensures the generated Secret exists with the correct content. It updates the DerivedSecret
// status after every attempt.
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
		log.Error(nil, "unable to fetch parent secret. NOTE: if you are referencing a secret cross-namespace, make sure it has the 'secretderiver.lightjack.de/allowCrossnamespaceReference=true' label!", "namespace", parentSecretRef.Namespace, "name", parentSecretRef.Name)
		if err := r.handleParentSecretNotFound(ctx, derivedSecret); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if parentSecret.Namespace != derivedSecret.Namespace && parentSecret.Labels["secretderiver.lightjack.de/allowCrossnamespaceReference"] != "true" {
		log.Error(nil, "unable to fetch parent secret. NOTE: if you are referencing a secret cross-namespace, make sure it has the 'secretderiver.lightjack.de/allowCrossnamespaceReference=true' label!", "namespace", parentSecretRef.Namespace, "name", parentSecretRef.Name)
		if err := r.handleParentSecretNotFound(ctx, derivedSecret); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
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

// createDerivedSecret creates the generated Secret if it does not exist, or updates it if its
// content differs from the expected derived value. Updates are skipped when the content is
// already correct to avoid unnecessary API writes.
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

// equalSecretData reports whether two Secret data maps have identical keys and values.
// Values are compared using constant-time comparison to avoid timing side channels.
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

// deriveValue produces a deterministic 32-byte secret using HKDF-SHA256.
// The HKDF input key material is sourceValue; the salt is "namespace/name" of the
// DerivedSecret. The output is returned as a base64url-encoded string (no padding).
func (r *DerivedSecretReconciler) deriveValue(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret, sourceValue []byte) ([]byte, error) {
	salt := []byte(derivedSecret.Namespace + "/" + derivedSecret.Name)
	hkdfReader := hkdf.New(sha256.New, sourceValue, salt, nil)

	rawValue := make([]byte, 32)
	_, err := io.ReadFull(hkdfReader, rawValue)
	if err != nil {
		return nil, fmt.Errorf("failed to derive value using HKDF: %w", err)
	}

	return []byte(base64.RawURLEncoding.EncodeToString(rawValue)), nil
}

// updateStatusReady sets the DerivedSecret status to Ready after a successful derivation.
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

// handleDerivationError records an Error status when HKDF derivation fails.
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

// handleEmptyValueInParent records an Error status when the referenced key exists in the
// parent secret but its value is empty.
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

// handleKeyNotFoundInParent records an Error status when the specified parentSecretKey does
// not exist in the parent secret.
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

// handleParentSecretNotFound records an Error status when the parent secret cannot be
// resolved — either because it does not exist or because a cross-namespace reference is
// missing the required opt-in label.
func (r *DerivedSecretReconciler) handleParentSecretNotFound(ctx context.Context, derivedSecret *secretderiverv1alpha1.DerivedSecret) error {
	derivedSecret.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             metav1.ConditionFalse,
			Reason:             "ParentSecretNotFound",
			Message:            "The specified parent secret does not exist. NOTE: If you are referencing a secret cross-namespace, make sure it has the 'secretderiver.lightjack.de/allowCrossnamespaceReference=true' label!",
			LastTransitionTime: metav1.Now(),
		},
	}
	derivedSecret.Status.Phase = "Error"
	if err := r.Status().Update(ctx, derivedSecret); err != nil {
		return err
	}
	return nil
}

// SetupWithManager registers the controller with the Manager. In addition to owning the
// generated Secrets, the controller watches all Secrets so that changes to a parent secret
// immediately trigger reconciliation of any DerivedSecret that references it.
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

// findDerivedSecretsForParent maps a Secret to the DerivedSecrets that reference it as their
// parent, so that those resources are re-queued whenever the parent secret changes.
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
