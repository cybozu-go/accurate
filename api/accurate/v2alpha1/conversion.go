package v2alpha1

import (
	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"k8s.io/apimachinery/pkg/conversion"
)

// Convert_v2_SubNamespaceSpec_To_v2alpha1_SubNamespaceSpec complements the generated conversion functions.
// Parent exists only in v2 and is intentionally dropped when converting to v2alpha1.
func Convert_v2_SubNamespaceSpec_To_v2alpha1_SubNamespaceSpec(in *accuratev2.SubNamespaceSpec, out *SubNamespaceSpec, s conversion.Scope) error {
	if err := autoConvert_v2_SubNamespaceSpec_To_v2alpha1_SubNamespaceSpec(in, out, s); err != nil {
		return err
	}

	// Drop in.Parent intentionally.
	return nil
}
