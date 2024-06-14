package v2alpha1

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
}
