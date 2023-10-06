package feature

import (
	"github.com/cybozu-go/accurate/pkg/config"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/component-base/featuregate"
)

// DisablePropagateGenerated will disable watch for owners with propagate-generated
// annotation. This could be useful to avoid Accurate from attempting to modify resources
// in namespaces that should be out-of-scope for Accurate.
const DisablePropagateGenerated featuregate.Feature = "DisablePropagateGenerated"

func init() {
	runtime.Must(config.DefaultMutableFeatureGate.Add(defaultFeatureGates))
}

var defaultFeatureGates = map[featuregate.Feature]featuregate.FeatureSpec{
	DisablePropagateGenerated: {Default: false, PreRelease: featuregate.Alpha},
}
