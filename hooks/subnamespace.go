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

//+kubebuilder:webhook:path=/validate-accurate-cybozu-com-v1-subnamespace,mutating=false,failurePolicy=fail,sideEffects=None,groups=accurate.cybozu.com,resources=subnamespaces,verbs=create;update,versions=v1,name=vsubnamespace.kb.io,admissionReviewVersions={v1,v1beta1}

type subNamespaceValidator struct {
	client.Client
	dec            *admission.Decoder
	namingPolicies []config.NamingPolicy
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

	ok, err := v.notMatchingNamingPolicy(ctx, sn)
	if err != nil {
		return admission.Errored(http.StatusInternalServerError, err)
	}
	if !ok {
		return admission.Denied(fmt.Sprintf("namespace %s is not match naming policies", ns.Name))
	}

	return admission.Allowed("")
}

func (v *subNamespaceValidator) notMatchingNamingPolicy(ctx context.Context, sn *accuratev1.SubNamespace) (bool, error) {
	for _, policy := range v.namingPolicies {
		matched, err := regexp.MatchString(policy.Root, sn.Namespace)
		if err != nil {
			return false, err
		}

		if matched {
			matched, err := regexp.MatchString(policy.Match, sn.Name)
			if err != nil {
				return false, err
			}
			if !matched {
				return false, nil
			}
		}
	}

	return true, nil
}

// SetupSubNamespaceWebhook registers the webhooks for SubNamespace
func SetupSubNamespaceWebhook(mgr manager.Manager, dec *admission.Decoder, namingPolicies []config.NamingPolicy) {
	serv := mgr.GetWebhookServer()

	m := &subNamespaceMutator{
		dec: dec,
	}
	serv.Register("/mutate-accurate-cybozu-com-v1-subnamespace", &webhook.Admission{Handler: m})

	v := &subNamespaceValidator{
		Client:         mgr.GetClient(),
		dec:            dec,
		namingPolicies: namingPolicies,
	}
	serv.Register("/validate-accurate-cybozu-com-v1-subnamespace", &webhook.Admission{Handler: v})
}
