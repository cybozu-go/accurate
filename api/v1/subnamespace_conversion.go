package v1

import (
	"encoding/json"
	"fmt"
	"strconv"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/meta"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this SubNamespace to the Hub version (v2alpha1).
func (src *SubNamespace) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*accuratev2alpha1.SubNamespace)

	logger := getConversionLogger(src).WithValues(
		"source", GroupVersion.Version,
		"destination", GroupVersion.Version,
	)
	logger.V(5).Info("converting")

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Labels = src.Spec.Labels

	// Restore info from annotations to ensure conversions are lossy-less.
	// Delete annotation after processing it to avoid polluting converted resource.
	if v, ok := dst.Annotations[constants.AnnObservedGeneration]; ok {
		obsGen, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("error converting %q to int64 from annotation %s", v, constants.AnnObservedGeneration)
		}
		dst.Status.ObservedGeneration = obsGen

		delete(dst.Annotations, constants.AnnObservedGeneration)
	}
	if conds, ok := dst.Annotations[constants.AnnConditions]; ok {
		err := json.Unmarshal([]byte(conds), &dst.Status.Conditions)
		if err != nil {
			return fmt.Errorf("error unmarshalling JSON from annotation %s", constants.AnnConditions)
		}

		delete(dst.Annotations, constants.AnnConditions)
	}

	return nil
}

// ConvertFrom converts from the Hub version (v2alpha1) to this version.
func (dst *SubNamespace) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*accuratev2alpha1.SubNamespace)

	logger := getConversionLogger(src).WithValues(
		"source", GroupVersion.Version,
		"destination", GroupVersion.Version,
	)
	logger.V(5).Info("converting")

	dst.ObjectMeta = src.ObjectMeta
	dst.Spec.Annotations = src.Spec.Annotations
	dst.Spec.Labels = src.Spec.Labels

	switch {
	case meta.IsStatusConditionTrue(src.Status.Conditions, string(kstatus.ConditionStalled)):
		dst.Status = SubNamespaceConflict
	case src.Status.ObservedGeneration == 0:
		// SubNamespace has never been reconciled.
	case src.Status.ObservedGeneration == src.Generation && len(src.Status.Conditions) == 0:
		dst.Status = SubNamespaceOK
	default:
		// SubNamespace is in some transitional state, not possible to represent in v1 status.
		// An unset value is probably our best option.
	}

	// Store info in annotations to ensure conversions are lossy-less.
	if dst.Annotations == nil {
		dst.Annotations = make(map[string]string)
	}
	if src.Status.ObservedGeneration != 0 {
		dst.Annotations[constants.AnnObservedGeneration] = strconv.FormatInt(src.Status.ObservedGeneration, 10)
	}
	if len(src.Status.Conditions) > 0 {
		buf, err := json.Marshal(src.Status.Conditions)
		if err != nil {
			return fmt.Errorf("error marshalling conditions to JSON")
		}
		dst.Annotations[constants.AnnConditions] = string(buf)
	}
	if len(dst.Annotations) == 0 {
		dst.Annotations = nil
	}

	return nil
}

func getConversionLogger(obj client.Object) logr.Logger {
	return ctrl.Log.WithName("conversion").WithValues(
		"kind", "SubNamespace",
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)
}
