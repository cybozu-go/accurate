package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
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
func upgradeManagedFields(ctx context.Context, c client.Client, obj client.Object, opts ...csaupgrade.Option) error {
	patch, err := csaupgrade.UpgradeManagedFieldsPatch(obj, sets.New(string(fieldOwner)), string(fieldOwner), opts...)
	if err != nil {
		return err
	}
	if patch != nil {
		return c.Patch(ctx, obj, client.RawPatch(types.JSONPatchType, patch))
	}
	// No work to be done - already upgraded
	return nil
}
