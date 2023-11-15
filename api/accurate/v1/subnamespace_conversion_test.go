package v1

import (
	"testing"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
)

func TestSubNamespace_ConvertFrom(t *testing.T) {
	tests := map[string]struct {
		src       *accuratev2alpha1.SubNamespace
		expStatus SubNamespaceStatus
		wantErr   bool
	}{
		"if SubNamespace has never been reconciled, status should have zero-value": {
			src: newSubNamespaceWithStatus(0, 0),
		},
		"if SubNamespace spec is updated, but not yet reconciled, status should have zero-value": {
			src: newSubNamespaceWithStatus(2, 1),
		},
		"if SubNamespace is reconciled successfully, status should be ok": {
			src:       newSubNamespaceWithStatus(1, 1),
			expStatus: SubNamespaceOK,
		},
		"if SubNamespace is reconciled with errors, status should be conflict": {
			src:       newSubNamespaceWithStatus(1, 1, newStalledCondition()),
			expStatus: SubNamespaceConflict,
		},
	}
	for n, tt := range tests {
		t.Run(n, func(t *testing.T) {
			dst := &SubNamespace{}
			if err := dst.ConvertFrom(tt.src); (err != nil) != tt.wantErr {
				t.Errorf("ConvertFrom() error = %v, wantErr %v", err, tt.wantErr)
			}
			if dst.Status != tt.expStatus {
				t.Errorf("ConvertFrom() status = %q, expStatus %q", dst.Status, tt.expStatus)
			}
		})
	}
}

func newSubNamespaceWithStatus(gen, obsGen int, conds ...metav1.Condition) *accuratev2alpha1.SubNamespace {
	subNS := &accuratev2alpha1.SubNamespace{}
	subNS.Generation = int64(gen)
	subNS.Status.ObservedGeneration = int64(obsGen)
	subNS.Status.Conditions = conds
	return subNS
}

func newStalledCondition() metav1.Condition {
	return metav1.Condition{
		Type:   string(kstatus.ConditionStalled),
		Status: metav1.ConditionTrue,
	}
}
