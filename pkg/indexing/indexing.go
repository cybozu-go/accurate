package indexing

import (
	"context"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"github.com/cybozu-go/accurate/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupIndexForResource sets up an indexer for a watched resource.
func SetupIndexForResource(ctx context.Context, mgr manager.Manager, res client.Object) error {
	return mgr.GetFieldIndexer().IndexField(ctx, res, constants.PropagateKey, func(rawObj client.Object) []string {
		val := rawObj.GetAnnotations()[constants.AnnPropagate]
		if val == "" {
			return nil
		}
		return []string{val, constants.PropagateAny}
	})
}

// SetupIndexForNamespace sets up indexers for namespaces.
func SetupIndexForNamespace(ctx context.Context, mgr manager.Manager) error {
	ns := &corev1.Namespace{}
	err := mgr.GetFieldIndexer().IndexField(ctx, ns, constants.NamespaceParentKey, func(rawObj client.Object) []string {
		parent := rawObj.GetLabels()[constants.LabelParent]
		if parent == "" {
			return nil
		}
		return []string{parent}
	})
	if err != nil {
		return err
	}

	return mgr.GetFieldIndexer().IndexField(context.Background(), ns, constants.NamespaceTemplateKey, func(rawObj client.Object) []string {
		tmpl := rawObj.GetLabels()[constants.LabelTemplate]
		if tmpl == "" {
			return nil
		}
		return []string{tmpl}
	})
}

// SetupIndexForSubNamespace sets up indexers for subnamespaces.
func SetupIndexForSubNamespace(ctx context.Context, mgr manager.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &accuratev2.SubNamespace{}, constants.SubNamespaceNameKey, func(rawObj client.Object) []string {
		return []string{rawObj.GetName()}
	})
}
