package hooks

import (
	"context"
	"net/http"

	"github.com/cybozu-go/innu/pkg/constants"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-v1-namespace,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=namespaces,verbs=create;update;delete,versions=v1,name=vnamespace.kb.io,admissionReviewVersions={v1,v1beta1}

type namespaceValidator struct {
	client.Client
	dec *admission.Decoder
}

var _ admission.Handler = &namespaceValidator{}

// Validate Namespace to prevent the following problems:
//
// - Circular references among sub-namespaces.
// - Deleting `innu.cybozu.com/root` label from root namespaces having one or more sub-namespaces.
// - Dangling sub-namespaces (sub-namespaces whose parent is missing).
// - Creating a sub-namespace under a non-root and non-sub- namespace.
// - Changing a sub-namespace that has child sub-namespaces to a non-root namespace.
func (v *namespaceValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	switch req.Operation {
	case admissionv1.Create:
		ns := &corev1.Namespace{}
		if err := v.dec.Decode(req, ns); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		return v.handleCreate(ctx, ns)

	case admissionv1.Update:
		nsNew := &corev1.Namespace{}
		if err := v.dec.Decode(req, nsNew); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		nsOld := &corev1.Namespace{}
		if err := v.dec.DecodeRaw(req.OldObject, nsOld); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		return v.handleUpdate(ctx, nsNew, nsOld)

	case admissionv1.Delete:
		ns := &corev1.Namespace{}
		if err := v.dec.DecodeRaw(req.OldObject, ns); err != nil {
			return admission.Errored(http.StatusBadRequest, err)
		}
		return v.handleDelete(ctx, ns)
	}
	return admission.Denied("unknown operation: " + string(req.Operation))
}

func (v *namespaceValidator) checkParent(ctx context.Context, p string) admission.Response {
	if p == "" {
		return admission.Allowed("")
	}

	parent := &corev1.Namespace{}
	err := v.Get(ctx, client.ObjectKey{Name: p}, parent)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return admission.Denied("parent namespace does not exist: " + p)
		}
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if parent.Labels[constants.LabelRoot] == "true" {
		return admission.Allowed("")
	}
	if parent.Labels[constants.LabelParent] != "" {
		return admission.Allowed("")
	}

	return admission.Denied("parent must be a root or sub-namespace")
}

func (v *namespaceValidator) handleCreate(ctx context.Context, ns *corev1.Namespace) admission.Response {
	p := ns.Labels[constants.LabelParent]
	if p != "" && ns.Name == p {
		return admission.Denied("circular reference is not permitted")
	}
	if p != "" && ns.Labels[constants.LabelTemplate] != "" {
		return admission.Denied("a sub-namespace cannot have a template")
	}
	return v.checkParent(ctx, p)
}

func (v *namespaceValidator) handleUpdate(ctx context.Context, nsNew, nsOld *corev1.Namespace) admission.Response {
	m := map[string]bool{nsNew.Name: true}
	p := nsNew.Labels[constants.LabelParent]
	for pp := p; pp != ""; {
		if m[pp] {
			return admission.Denied("circular reference is not permitted")
		}
		parent := &corev1.Namespace{}
		if err := v.Get(ctx, client.ObjectKey{Name: pp}, parent); err != nil {
			if apierrors.IsNotFound(err) {
				return admission.Denied("parent namespace does not exist: " + p)
			}
			return admission.Errored(http.StatusInternalServerError, err)
		}
		m[pp] = true
		pp = parent.Labels[constants.LabelParent]
	}

	if p != "" && nsNew.Labels[constants.LabelTemplate] != "" {
		return admission.Denied("a sub-namespace cannot have a template")
	}

	// those who have sub-namespaces should be either root or a sub-namespace.
	if p == "" && nsNew.Labels[constants.LabelRoot] != "true" {
		children := &corev1.NamespaceList{}
		if err := v.List(ctx, children, client.MatchingFields{constants.NamespaceParentKey: nsNew.Name}); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if len(children.Items) > 0 {
			return admission.Denied("sub-namespaces exist")
		}
	}

	if p != nsOld.Labels[constants.LabelParent] {
		return v.checkParent(ctx, p)
	}

	return admission.Allowed("")
}

func (v *namespaceValidator) handleDelete(ctx context.Context, ns *corev1.Namespace) admission.Response {
	if ns.Labels[constants.LabelRoot] != "true" && ns.Labels[constants.LabelParent] == "" {
		return admission.Allowed("")
	}

	children := &corev1.NamespaceList{}
	if err := v.List(ctx, children, client.MatchingFields{constants.NamespaceParentKey: ns.Name}); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if len(children.Items) > 0 {
		return admission.Denied("sub-namespaces exist")
	}

	return admission.Allowed("")
}

// SetupNamespaceWebhook registers the webhook for Namespace
func SetupNamespaceWebhook(mgr manager.Manager, dec *admission.Decoder) {
	v := &namespaceValidator{
		Client: mgr.GetClient(),
		dec:    dec,
	}
	serv := mgr.GetWebhookServer()
	serv.Register("/validate-v1-namespace", &webhook.Admission{Handler: v})
}
