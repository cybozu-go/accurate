package controllers

import (
	"context"
	"encoding/json"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	accuratev2alpha1ac "github.com/cybozu-go/accurate/internal/applyconfigurations/accurate/v2alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	corev1ac "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/util/csaupgrade"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	fieldOwner client.FieldOwner = "accurate-controller"
)

// TODO(migration): This code could be removed after a couple of releases.
// upgradeManagedFields is a migration function that migrates the ownership of
// fields from the Update operation to the Apply operation. This is required
// to ensure that the apply operations will also remove fields that were
// set by the Update operation.
func upgradeManagedFields(ctx context.Context, c client.Client, obj client.Object) error {
	patch, err := csaupgrade.UpgradeManagedFieldsPatch(obj, sets.New(string(fieldOwner)), string(fieldOwner))
	if err != nil {
		return err
	}
	if patch != nil {
		return c.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, patch))
	}
	// No work to be done - already upgraded
	return nil
}

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
