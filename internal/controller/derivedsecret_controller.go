package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

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
	_ = logf.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DerivedSecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&secretderiverv1alpha1.DerivedSecret{}).
		Named("derivedsecret").
		Complete(r)
}
