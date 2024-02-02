package controllers

import (
	"encoding/json"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	accuratev2alpha1ac "github.com/cybozu-go/accurate/internal/applyconfigurations/accurate/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	fieldOwner client.FieldOwner = "accurate-controller"
)

func newSubNamespacePatch(ac *accuratev2alpha1ac.SubNamespaceApplyConfiguration) (*accuratev2alpha1.SubNamespace, client.Patch, error) {
	sn := &accuratev2alpha1.SubNamespace{}
	sn.Name = *ac.Name
	sn.Namespace = *ac.Namespace

	return sn, applyPatch{ac}, nil
}

func newNamespacePatch(ac *corev1ac.NamespaceApplyConfiguration) (*corev1.Namespace, client.Patch, error) {
	ns := &corev1.Namespace{}
	ns.Name = *ac.Name

	return ns, applyPatch{ac}, nil
}

type applyPatch struct {
	// must use any type until apply configurations implements a common interface
	patch any
}

func (p applyPatch) Type() types.PatchType {
	return types.ApplyPatchType
}

func (p applyPatch) Data(_ client.Object) ([]byte, error) {
	return json.Marshal(p.patch)
}
