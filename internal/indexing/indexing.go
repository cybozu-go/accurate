package indexing

import (
	"context"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// SetupIndexForResource sets up an indexer for a watched resource.
func SetupIndexForResource(ctx context.Context, mgr manager.Manager, res client.Object) error {
	return mgr.GetFieldIndexer().IndexField(ctx, res, PropagateKey, func(rawObj client.Object) []string {
		val := rawObj.GetAnnotations()[accuratev2.AnnPropagate]
		if val == "" {
			return nil
		}
		return []string{val, PropagateAny}
	})
}

// SetupIndexForNamespace sets up indexers for namespaces.
func SetupIndexForNamespace(ctx context.Context, mgr manager.Manager) error {
	ns := &corev1.Namespace{}
	err := mgr.GetFieldIndexer().IndexField(ctx, ns, NamespaceParentKey, func(rawObj client.Object) []string {
		parent := rawObj.GetLabels()[accuratev2.LabelParent]
		if parent == "" {
			return nil
		}
		return []string{parent}
	})
	if err != nil {
		return err
	}

	return mgr.GetFieldIndexer().IndexField(ctx, ns, NamespaceTemplateKey, func(rawObj client.Object) []string {
		tmpl := rawObj.GetLabels()[accuratev2.LabelTemplate]
		if tmpl == "" {
			return nil
		}
		return []string{tmpl}
	})
}

// SetupIndexForSubNamespace sets up indexers for subnamespaces.
func SetupIndexForSubNamespace(ctx context.Context, mgr manager.Manager) error {
	return mgr.GetFieldIndexer().IndexField(ctx, &accuratev2.SubNamespace{}, SubNamespaceNameKey, func(rawObj client.Object) []string {
		return []string{rawObj.GetName()}
	})
}
