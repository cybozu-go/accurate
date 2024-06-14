package v2alpha1

import (
	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
)

// ConvertTo converts this SubNamespace to the Hub version (v2alpha1).
func (src *SubNamespace) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*accuratev2.SubNamespace)

	logger := getConversionLogger(src).WithValues(
		"source", SchemeGroupVersion.Version,
		"destination", accuratev2.SchemeGroupVersion.Version,
	)
	logger.V(5).Info("converting")

	return Convert_v2alpha1_SubNamespace_To_v2_SubNamespace(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v2alpha1) to this version.
func (dst *SubNamespace) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*accuratev2.SubNamespace)

	logger := getConversionLogger(src).WithValues(
		"source", accuratev2.SchemeGroupVersion.Version,
		"destination", SchemeGroupVersion.Version,
	)
	logger.V(5).Info("converting")

	return Convert_v2_SubNamespace_To_v2alpha1_SubNamespace(src, dst, nil)
}

func getConversionLogger(obj client.Object) logr.Logger {
	return ctrl.Log.WithName("conversion").WithValues(
		"kind", "SubNamespace",
		"namespace", obj.GetNamespace(),
		"name", obj.GetName(),
	)
}
