package hooks

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"

	accuratev1 "github.com/cybozu-go/accurate/api/v1"
	"github.com/cybozu-go/accurate/pkg/config"
	"github.com/cybozu-go/accurate/pkg/constants"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/mutate-accurate-cybozu-com-v1-subnamespace,mutating=true,failurePolicy=fail,sideEffects=None,groups=accurate.cybozu.com,resources=subnamespaces,verbs=create;update,versions=v1,name=msubnamespace.kb.io,admissionReviewVersions={v1,v1beta1}

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

type NamingPolicyRegexp struct {
	Root  *regexp.Regexp
	Match *regexp.Regexp
}

//+kubebuilder:webhook:path=/validate-accurate-cybozu-com-v1-subnamespace,mutating=false,failurePolicy=fail,sideEffects=None,groups=accurate.cybozu.com,resources=subnamespaces,verbs=create;update,versions=v1,name=vsubnamespace.kb.io,admissionReviewVersions={v1,v1beta1}

type subNamespaceValidator struct {
	client.Client
	dec            *admission.Decoder
	namingPolicies []NamingPolicyRegexp
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

	root, err := v.getRootNamespace(ctx, ns)
	if err != nil {
		return admission.Denied(err.Error())
	}
	ok, err := v.notMatchingNamingPolicy(ctx, sn.Name, root.Name)
	if !ok {
		return admission.Denied(fmt.Sprintf("namespace %s is not match naming policies: %s", ns.Name, err.Error()))
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

func (v *subNamespaceValidator) notMatchingNamingPolicy(ctx context.Context, ns, root string) (bool, error) {
	for _, policy := range v.namingPolicies {
		if policy.Root.MatchString(root) {
			if !policy.Match.MatchString(ns) {
				return false, fmt.Errorf("namespace name - target=%s root=%s denied policy - root: %s match: %s", ns, root, policy.Root, policy.Match)
			}
		}
	}
	return true, nil
}

// SetupSubNamespaceWebhook registers the webhooks for SubNamespace
func SetupSubNamespaceWebhook(mgr manager.Manager, dec *admission.Decoder, namingPolicies []config.NamingPolicy) error {
	serv := mgr.GetWebhookServer()

	m := &subNamespaceMutator{
		dec: dec,
	}
	serv.Register("/mutate-accurate-cybozu-com-v1-subnamespace", &webhook.Admission{Handler: m})

	namingPolicyRegexps, err := compileNamingPolicies(namingPolicies)
	if err != nil {
		return err
	}

	v := &subNamespaceValidator{
		Client:         mgr.GetClient(),
		dec:            dec,
		namingPolicies: namingPolicyRegexps,
	}
	serv.Register("/validate-accurate-cybozu-com-v1-subnamespace", &webhook.Admission{Handler: v})
	return nil
}

func compileNamingPolicies(namingPolicies []config.NamingPolicy) ([]NamingPolicyRegexp, error) {
	var result []NamingPolicyRegexp
	for _, policy := range namingPolicies {
		root, err := regexp.Compile(policy.Root)
		if err != nil {
			return nil, err
		}

		match, err := regexp.Compile(policy.Match)
		if err != nil {
			return nil, err
		}

		result = append(result, NamingPolicyRegexp{Root: root, Match: match})
	}
	return result, nil
}
