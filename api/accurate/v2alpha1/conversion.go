package v2alpha1

import (
	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"github.com/cybozu-go/accurate/pkg/constants"
	"k8s.io/apimachinery/pkg/conversion"
)

// Convert_v2alpha1_SubNamespace_To_v2_SubNamespace complements the generated conversion functions
// since Parent exists only in v2 and must be restored from the annotation stored during down-conversion.
func Convert_v2alpha1_SubNamespace_To_v2_SubNamespace(in *SubNamespace, out *accuratev2.SubNamespace, s conversion.Scope) error {
	if err := autoConvert_v2alpha1_SubNamespace_To_v2_SubNamespace(in, out, s); err != nil {
		return err
	}

	// Restore info from annotations to ensure conversions are lossy-less.
	// Delete annotation after processing it to avoid polluting converted resource.
	if v, ok := out.Annotations[constants.AnnParent]; ok {
		out.Spec.Parent = v

		delete(out.Annotations, constants.AnnParent)
	}
	if len(out.Annotations) == 0 {
		out.Annotations = nil
	}
	return nil
}

// Convert_v2_SubNamespace_To_v2alpha1_SubNamespace complements the generated conversion functions
// since Parent exists only in v2 and is stored in an annotation to ensure conversions are lossy-less.
func Convert_v2_SubNamespace_To_v2alpha1_SubNamespace(in *accuratev2.SubNamespace, out *SubNamespace, s conversion.Scope) error {
	if err := autoConvert_v2_SubNamespace_To_v2alpha1_SubNamespace(in, out, s); err != nil {
		return err
	}

	// Store info in annotations to ensure conversions are lossy-less.
	if in.Spec.Parent != "" {
		if out.Annotations == nil {
			out.Annotations = make(map[string]string)
		}
		out.Annotations[constants.AnnParent] = in.Spec.Parent
	}
	return nil
}

// Convert_v2_SubNamespaceSpec_To_v2alpha1_SubNamespaceSpec complements the generated conversion functions.
// Parent exists only in v2 and is intentionally dropped when converting to v2alpha1.
func Convert_v2_SubNamespaceSpec_To_v2alpha1_SubNamespaceSpec(in *accuratev2.SubNamespaceSpec, out *SubNamespaceSpec, s conversion.Scope) error {
	if err := autoConvert_v2_SubNamespaceSpec_To_v2alpha1_SubNamespaceSpec(in, out, s); err != nil {
		return err
	}

	// Drop in.Parent intentionally.
	return nil
}
