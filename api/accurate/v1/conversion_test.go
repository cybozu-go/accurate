package v1

import (
	"testing"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	utilconversion "github.com/cybozu-go/accurate/internal/util/conversion"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	"sigs.k8s.io/randfill"
)

func TestFuzzyConversion(t *testing.T) {
	t.Run("for SubNamespace", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:   &accuratev2.SubNamespace{},
		Spoke: &SubNamespace{},
		HubAfterMutation: func(hub conversion.Hub) {
			if ns, ok := hub.(*accuratev2.SubNamespace); ok {
				ns.TypeMeta.Kind = ""
				ns.TypeMeta.APIVersion = ""
			}
		},
		SpokeAfterMutation: func(spoke conversion.Convertible) {
			if ns, ok := spoke.(*SubNamespace); ok {
				ns.TypeMeta.Kind = ""
				ns.TypeMeta.APIVersion = ""
				ns.Status = ""
			}
		},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{SubNamespaceStatusFuzzFunc},
	}))
}

func SubNamespaceStatusFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		SubNamespaceStatusFuzzer,
	}
}

func SubNamespaceStatusFuzzer(in *SubNamespace, c randfill.Continue) {
	c.Fill(in)

	// The status is just a string in v1, and the controller is the sole actor updating status.
	// As long as we make the controller reconcile v2, and also makes it the stored version,
	// we will never need to convert status from v1 to v2.
	in.Status = ""
}
