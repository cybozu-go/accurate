package v1

import (
	"context"
	"fmt"

	"github.com/cybozu-go/innu/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var k8sclient client.Client

func SetClientForWebhook(c client.Client) {
	k8sclient = c
}

func (r *SubNamespace) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

//+kubebuilder:webhook:path=/mutate-innu-cybozu-com-v1-subnamespace,mutating=true,failurePolicy=fail,sideEffects=None,groups=innu.cybozu.com,resources=subnamespaces,verbs=create;update,versions=v1,name=msubnamespace.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Defaulter = &SubNamespace{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *SubNamespace) Default() {
	controllerutil.AddFinalizer(r, constants.Finalizer)
}

//+kubebuilder:webhook:path=/validate-innu-cybozu-com-v1-subnamespace,mutating=false,failurePolicy=fail,sideEffects=None,groups=innu.cybozu.com,resources=subnamespaces,verbs=create;update,versions=v1,name=vsubnamespace.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &SubNamespace{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SubNamespace) ValidateCreate() error {
	ns := &corev1.Namespace{}
	if err := k8sclient.Get(context.Background(), client.ObjectKey{Name: r.Namespace}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", r.Namespace, err)
	}
	if ns.Labels[constants.LabelRoot] == "true" || ns.Labels[constants.LabelParent] != "" {
		return nil
	}

	return fmt.Errorf("namespace %s is neither root nor sub namespace", r.Namespace)
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SubNamespace) ValidateUpdate(old runtime.Object) error {
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SubNamespace) ValidateDelete() error {
	return nil
}
