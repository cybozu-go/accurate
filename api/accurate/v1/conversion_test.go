package v1

import (
	"testing"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	utilconversion "github.com/cybozu-go/accurate/internal/util/conversion"
	fuzz "github.com/google/gofuzz"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

func TestFuzzyConversion(t *testing.T) {
	t.Run("for SubNamespace", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &accuratev2.SubNamespace{},
		Spoke:       &SubNamespace{},
		FuzzerFuncs: []fuzzer.FuzzerFuncs{SubNamespaceStatusFuzzFunc},
	}))
}

func SubNamespaceStatusFuzzFunc(_ runtimeserializer.CodecFactory) []interface{} {
	return []interface{}{
		SubNamespaceStatusFuzzer,
	}
}

func SubNamespaceStatusFuzzer(in *SubNamespace, c fuzz.Continue) {
	c.FuzzNoCustom(in)

	// The status is just a string in v1, and the controller is the sole actor updating status.
	// As long as we make the controller reconcile v2, and also makes it the stored version,
	// we will never need to convert status from v1 to v2.
	in.Status = ""
}
