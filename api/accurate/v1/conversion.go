package v1

import (
	"encoding/json"
	"fmt"
	"strconv"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/constants"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/conversion"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

// Convert_v1_SubNamespace_To_v2alpha1_SubNamespace complements the generated conversion functions since status needs special handling
func Convert_v1_SubNamespace_To_v2alpha1_SubNamespace(in *SubNamespace, out *accuratev2alpha1.SubNamespace, s conversion.Scope) error {
	if err := autoConvert_v1_SubNamespace_To_v2alpha1_SubNamespace(in, out, s); err != nil {
		return err
	}

	// Restore info from annotations to ensure conversions are lossy-less.
	// Delete annotation after processing it to avoid polluting converted resource.
	if v, ok := out.Annotations[constants.AnnObservedGeneration]; ok {
		obsGen, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("error converting %q to int64 from annotation %s", v, constants.AnnObservedGeneration)
		}
		out.Status.ObservedGeneration = obsGen

		delete(out.Annotations, constants.AnnObservedGeneration)
	}
	if conds, ok := out.Annotations[constants.AnnConditions]; ok {
		err := json.Unmarshal([]byte(conds), &out.Status.Conditions)
		if err != nil {
			return fmt.Errorf("error unmarshalling JSON from annotation %s", constants.AnnConditions)
		}

		delete(out.Annotations, constants.AnnConditions)
	}
	return nil
}

// Convert_v2alpha1_SubNamespace_To_v1_SubNamespace complements the generated conversion functions since status needs special handling
func Convert_v2alpha1_SubNamespace_To_v1_SubNamespace(in *accuratev2alpha1.SubNamespace, out *SubNamespace, s conversion.Scope) error {
	if err := autoConvert_v2alpha1_SubNamespace_To_v1_SubNamespace(in, out, s); err != nil {
		return err
	}

	switch {
	case meta.IsStatusConditionTrue(in.Status.Conditions, string(kstatus.ConditionStalled)):
		out.Status = SubNamespaceConflict
	case in.Status.ObservedGeneration == 0:
		// SubNamespace has never been reconciled.
	case in.Status.ObservedGeneration == in.Generation && len(in.Status.Conditions) == 0:
		out.Status = SubNamespaceOK
	default:
		// SubNamespace is in some transitional state, not possible to represent in v1 status.
		// An unset value is probably our best option.
	}

	// Store info in annotations to ensure conversions are lossy-less.
	if out.Annotations == nil {
		out.Annotations = make(map[string]string)
	}
	if in.Status.ObservedGeneration != 0 {
		out.Annotations[constants.AnnObservedGeneration] = strconv.FormatInt(in.Status.ObservedGeneration, 10)
	}
	if len(in.Status.Conditions) > 0 {
		buf, err := json.Marshal(in.Status.Conditions)
		if err != nil {
			return fmt.Errorf("error marshalling conditions to JSON")
		}
		out.Annotations[constants.AnnConditions] = string(buf)
	}
	if len(out.Annotations) == 0 {
		out.Annotations = nil
	}
	return nil
}
