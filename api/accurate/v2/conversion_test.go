package v2

import (
	"testing"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	utilconversion "github.com/cybozu-go/accurate/internal/util/conversion"
	fuzz "github.com/google/gofuzz"
	"k8s.io/apimachinery/pkg/api/apitesting/fuzzer"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"
)

func TestFuzzyConversion(t *testing.T) {
	t.Run("for SubNamespace", utilconversion.FuzzTestFunc(utilconversion.FuzzTestFuncInput{
		Hub:         &accuratev2alpha1.SubNamespace{},
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
}
