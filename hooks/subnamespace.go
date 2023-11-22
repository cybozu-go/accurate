package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	accuratev1 "github.com/cybozu-go/accurate/api/accurate/v1"
	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/config"
	"github.com/cybozu-go/accurate/pkg/constants"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1annotationvalidation "k8s.io/apimachinery/pkg/api/validation"
	v1labelvalidation "k8s.io/apimachinery/pkg/apis/meta/v1/validation"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-accurate-cybozu-com-v1-subnamespace,mutating=true,failurePolicy=fail,sideEffects=None,groups=accurate.cybozu.com,resources=subnamespaces,verbs=create;update,versions=v1,matchPolicy=Equivalent,name=subnamespace.accurate.cybozu.io,admissionReviewVersions={v1}

type subNamespaceMutator struct {
	dec *admission.Decoder
}

var _ admission.Handler = &subNamespaceMutator{}

func (m *subNamespaceMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create {
		return admission.Allowed("")
	}

	sn := &accuratev1.SubNamespace{}
	if err := m.dec.Decode(req, sn); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	sn.Finalizers = []string{constants.Finalizer}
	data, err := json.Marshal(sn)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	return admission.PatchResponseFromRaw(req.Object.Raw, data)
}

//+kubebuilder:webhook:path=/validate-accurate-cybozu-com-v1-subnamespace,mutating=false,failurePolicy=fail,sideEffects=None,groups=accurate.cybozu.com,resources=subnamespaces,verbs=create;update,versions=v1,matchPolicy=Equivalent,name=vsubnamespace.kb.io,admissionReviewVersions={v1}

type subNamespaceValidator struct {
	client.Client
	dec            *admission.Decoder
	namingPolicies []config.NamingPolicyRegexp
}

var _ admission.Handler = &subNamespaceValidator{}

func (v *subNamespaceValidator) Handle(ctx context.Context, req admission.Request) admission.Response {
	if req.Operation != admissionv1.Create {
		return admission.Allowed("")
	}

	sn := &accuratev1.SubNamespace{}
	if err := v.dec.Decode(req, sn); err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	ns := &corev1.Namespace{}
	if err := v.Get(ctx, client.ObjectKey{Name: req.Namespace}, ns); err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}

	if ns.Labels[constants.LabelType] != constants.NSTypeRoot && ns.Labels[constants.LabelParent] == "" {
		return admission.Denied(fmt.Sprintf("namespace %s is neither a root nor a sub namespace", ns.Name))
	}

	var allErrs field.ErrorList
	allErrs = append(allErrs, v1labelvalidation.ValidateLabels(sn.Spec.Labels, field.NewPath("spec", "labels"))...)
	allErrs = append(allErrs, v1annotationvalidation.ValidateAnnotations(sn.Spec.Annotations, field.NewPath("spec", "annotations"))...)
	if len(allErrs) != 0 {
		return admission.Denied(allErrs.ToAggregate().Error())
	}

	root, err := v.getRootNamespace(ctx, ns)
	if err != nil {
		return admission.Denied(err.Error())
	}
	ok, msg, err := v.notMatchingNamingPolicy(ctx, sn.Name, root.Name)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if !ok {
		return admission.Denied(fmt.Sprintf("namespace %s is not match naming policies: %s", ns.Name, msg))
	}
	return admission.Allowed("")
}

func (v *subNamespaceValidator) getRootNamespace(ctx context.Context, ns *corev1.Namespace) (*corev1.Namespace, error) {
	if ns.Labels[constants.LabelType] == constants.NSTypeRoot {
		return ns, nil
	}

	parent := &corev1.Namespace{}
	if err := v.Get(ctx, client.ObjectKey{Name: ns.Labels[constants.LabelParent]}, parent); err != nil {
		if !apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("failed to get namespace %s: %w", ns.Labels[constants.LabelParent], err)
		}
		return nil, fmt.Errorf("namespace %s is not found", ns.Labels[constants.LabelParent])
	}
	return v.getRootNamespace(ctx, parent)
}

func (v *subNamespaceValidator) notMatchingNamingPolicy(ctx context.Context, ns, root string) (bool, string, error) {
	for _, policy := range v.namingPolicies {
		matches := policy.Root.FindAllStringSubmatchIndex(root, -1)
		if len(matches) > 0 {
			m := []byte{}
			for _, match := range matches {
				m = policy.Root.ExpandString(m, policy.Match, root, match)
			}
			r, err := regexp.Compile(string(m))
			if err != nil {
				return false, "", fmt.Errorf("invalid naming policy: %w", err)
			}

			if !r.MatchString(ns) {
				return false, fmt.Sprintf("namespace - target=%s root=%s denied policy - root=%s match=%s", ns, root, policy.Root, policy.Match), nil
			}
		}
	}
	return true, "", nil
}

// SetupSubNamespaceWebhook registers the webhooks for SubNamespace
func SetupSubNamespaceWebhook(mgr manager.Manager, dec *admission.Decoder, namingPolicyRegexps []config.NamingPolicyRegexp) error {
	for _, s := range []runtime.Object{&accuratev1.SubNamespace{}, &accuratev2alpha1.SubNamespace{}} {
		err := ctrl.NewWebhookManagedBy(mgr).
			For(s).
			Complete()
		if err != nil {
			return err
		}
	}

	serv := mgr.GetWebhookServer()

	m := &subNamespaceMutator{
		dec: dec,
	}
	serv.Register("/mutate-accurate-cybozu-com-v1-subnamespace", &webhook.Admission{Handler: m})

	v := &subNamespaceValidator{
		Client:         mgr.GetClient(),
		dec:            dec,
		namingPolicies: namingPolicyRegexps,
	}
	serv.Register("/validate-accurate-cybozu-com-v1-subnamespace", &webhook.Admission{Handler: v})
	return nil
}
