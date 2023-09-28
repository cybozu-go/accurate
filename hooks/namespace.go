package hooks

import (
	"context"
	"fmt"
	"net/http"

	"github.com/cybozu-go/accurate/pkg/constants"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-v1-namespace,mutating=false,failurePolicy=fail,sideEffects=None,groups="",resources=namespaces,verbs=create;update;delete,versions=v1,name=namespace.accurate.cybozu.io,admissionReviewVersions={v1}

type namespaceValidator struct {
	client.Client
	dec *admission.Decoder
}

var _ admission.Handler = &namespaceValidator{}

// Validate Namespace to prevent the following problems:
//
// - Circular references among namespaces.
// - Allowing a sub-namespace to set a template.
// - Marking a sub-namespace as a root namespace.
// - Deleting `accurate.cybozu.com/type=root` label from root namespaces having one or more sub-namespaces.
// - Deleting `accurate.cybozu.com/type=template` label from template namespaces having one or more instance namespaces.
// - Dangling sub-namespaces (sub-namespaces whose parent namespace is missing).
// - Dangling instance namespaces (namespaces whose template namespace is missing).
// - Changing a sub-namespace to a non-root namespace when it has child sub-namespaces.
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

func (v *namespaceValidator) checkParent(ctx context.Context, name, typ string) *admission.Response {
	parent := &corev1.Namespace{}
	err := v.Get(ctx, client.ObjectKey{Name: name}, parent)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			resp := admission.Errored(http.StatusInternalServerError, err)
			return &resp
		}
		resp := admission.Denied("namespace does not exist: " + name)
		return &resp
	}

	if parent.Labels[constants.LabelType] == typ {
		return nil
	}
	if parent.Labels[constants.LabelParent] != "" {
		return nil
	}

	resp := admission.Denied(fmt.Sprintf("%s is not a valid %s namespace", name, typ))
	return &resp
}

func (v *namespaceValidator) handleCreate(ctx context.Context, ns *corev1.Namespace) admission.Response {
	if p := ns.Labels[constants.LabelParent]; p != "" {
		if ns.Name == p {
			return admission.Denied("circular reference is not permitted")
		}
		if _, ok := ns.Labels[constants.LabelTemplate]; ok {
			return admission.Denied("a sub-namespace cannot have a template")
		}
		if _, ok := ns.Labels[constants.LabelType]; ok {
			return admission.Denied("a sub-namespace cannot be a root or a template")
		}
		if resp := v.checkParent(ctx, p, constants.NSTypeRoot); resp != nil {
			return *resp
		}
	}
	if t := ns.Labels[constants.LabelTemplate]; t != "" {
		if ns.Name == t {
			return admission.Denied("circular reference is not permitted")
		}
		if resp := v.checkParent(ctx, t, constants.NSTypeTemplate); resp != nil {
			return *resp
		}
	}
	return admission.Allowed("")
}

func (v *namespaceValidator) getParent(ns *corev1.Namespace) string {
	if p := ns.Labels[constants.LabelParent]; p != "" {
		return p
	}
	return ns.Labels[constants.LabelTemplate]
}

func (v *namespaceValidator) handleUpdate(ctx context.Context, nsNew, nsOld *corev1.Namespace) admission.Response {
	m := map[string]bool{nsNew.Name: true}
	p := v.getParent(nsNew)
	for pp := p; pp != ""; {
		if m[pp] {
			return admission.Denied("circular reference is not permitted")
		}
		parent := &corev1.Namespace{}
		if err := v.Get(ctx, client.ObjectKey{Name: pp}, parent); err != nil {
			if apierrors.IsNotFound(err) {
				return admission.Denied("parent namespace does not exist: " + pp)
			}
			return admission.Errored(http.StatusInternalServerError, err)
		}
		m[pp] = true
		pp = v.getParent(parent)
	}

	oldType := nsOld.Labels[constants.LabelType]
	newType := nsNew.Labels[constants.LabelType]

	if oldType != newType {
		if oldType == constants.NSTypeRoot {
			children := &corev1.NamespaceList{}
			if err := v.List(ctx, children, client.MatchingFields{constants.NamespaceParentKey: nsNew.Name}); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
			if len(children.Items) > 0 {
				return admission.Denied("there are sub-namespaces under " + nsNew.Name)
			}
		}
		if oldType == constants.NSTypeTemplate {
			children := &corev1.NamespaceList{}
			if err := v.List(ctx, children, client.MatchingFields{constants.NamespaceTemplateKey: nsNew.Name}); err != nil {
				return admission.Errored(http.StatusInternalServerError, err)
			}
			if len(children.Items) > 0 {
				return admission.Denied("there are namespaces referencing " + nsNew.Name)
			}
		}
	}

	if p == "" && nsOld.Labels[constants.LabelParent] != "" && newType != constants.NSTypeRoot {
		children := &corev1.NamespaceList{}
		if err := v.List(ctx, children, client.MatchingFields{constants.NamespaceParentKey: nsNew.Name}); err != nil {
			return admission.Errored(http.StatusInternalServerError, err)
		}
		if len(children.Items) > 0 {
			return admission.Denied("there are sub-namespaces under " + nsNew.Name)
		}
	}

	return v.handleCreate(ctx, nsNew)
}

func (v *namespaceValidator) handleDelete(ctx context.Context, ns *corev1.Namespace) admission.Response {
	key := constants.NamespaceParentKey
	switch {
	case ns.Labels[constants.LabelType] == constants.NSTypeRoot:
	case ns.Labels[constants.LabelType] == constants.NSTypeTemplate:
		key = constants.NamespaceTemplateKey
	case ns.Labels[constants.LabelParent] != "":
	default:
		return admission.Allowed("")
	}

	children := &corev1.NamespaceList{}
	if err := v.List(ctx, children, client.MatchingFields{key: ns.Name}); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if len(children.Items) > 0 {
		return admission.Denied("child namespaces exist")
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
