package controllers

import (
	"context"
	"fmt"
	"path"
	"reflect"

	"github.com/cybozu-go/accurate/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	LabelKeys      []string
	AnnotationKeys []string
	Watched        []*unstructured.Unstructured
}

var _ reconcile.Reconciler = &NamespaceReconciler{}

// Reconcile implements reconcile.Reconciler interface.
func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, req.NamespacedName, ns); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if ns.DeletionTimestamp != nil {
		return ctrl.Result{}, nil
	}

	if err := r.reconcile(ctx, ns); err != nil {
		logger.Error(err, "failed to reconcile a namespace")
		return ctrl.Result{}, fmt.Errorf("failed to reconcile a namespace: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *NamespaceReconciler) reconcile(ctx context.Context, ns *corev1.Namespace) error {
	if parent, ok := ns.Labels[constants.LabelParent]; ok {
		return r.reconcileSubNamespace(ctx, ns, parent)
	}

	if tmpl, ok := ns.Labels[constants.LabelTemplate]; ok {
		if err := r.reconcileInstanceNamespace(ctx, ns, tmpl); err != nil {
			return err
		}
		// a template instance may also be a root or a template namespace, so don't return here.
	} else {
		if err := r.deletePropagatedResources(ctx, ns); err != nil {
			return err
		}
	}

	switch ns.Labels[constants.LabelType] {
	case constants.NSTypeTemplate:
		return r.reconcileTemplateNamespace(ctx, ns)
	case constants.NSTypeRoot:
		return r.reconcileRootNamespace(ctx, ns)
	}

	return nil
}

func (r *NamespaceReconciler) propagateMeta(ctx context.Context, ns, parent *corev1.Namespace) error {
	orig := ns.DeepCopy()
	for k, v := range parent.Labels {
		if ok := r.matchLabelKey(k); ok {
			ns.Labels[k] = v
		}
	}
	for k, v := range parent.Annotations {
		if ok := r.matchAnnotationKey(k); ok {
			if ns.Annotations == nil {
				ns.Annotations = make(map[string]string)
			}
			ns.Annotations[k] = v
		}
	}
	if !reflect.DeepEqual(ns.ObjectMeta, orig.ObjectMeta) {
		if err := r.Update(ctx, ns); err != nil {
			return fmt.Errorf("failed to propagate labels/annotations for namespace %s: %w", ns.Name, err)
		}
	}
	return nil
}

func (r *NamespaceReconciler) matchLabelKey(key string) bool {
	for _, l := range r.LabelKeys {
		// The glob pattern has been verified to be in the valid format when reading the config file.
		if ok, _ := path.Match(l, key); ok {
			return true
		}
	}

	return false
}

func (r *NamespaceReconciler) matchAnnotationKey(key string) bool {
	for _, a := range r.AnnotationKeys {
		// The glob pattern has been verified to be in the valid format when reading the config file.
		if ok, _ := path.Match(a, key); ok {
			return true
		}
	}

	return false
}

func (r *NamespaceReconciler) propagateResource(ctx context.Context, res *unstructured.Unstructured, parent, ns string) error {
	logger := log.FromContext(ctx)

	gvk := res.GroupVersionKind()
	gvkStr := gvk.String()
	gvk.Kind = gvk.Kind + "List"
	l := &unstructured.UnstructuredList{}
	l.SetGroupVersionKind(gvk)

	cl := l.DeepCopy()
	if err := r.List(ctx, cl, client.MatchingFields{constants.PropagateKey: constants.PropagateCreate}, client.InNamespace(parent)); err != nil {
		return fmt.Errorf("failed to list %s in %s with propagate=create: %w", gvkStr, parent, err)
	}
	for i := range cl.Items {
		pres := &cl.Items[i]
		if err := r.propagateCreate(ctx, pres, ns); err != nil {
			return fmt.Errorf("failed to propagate resource %s/%s of %s with propagate=create: %w", ns, pres.GetName(), gvkStr, err)
		}
	}

	ul := l.DeepCopy()
	if err := r.List(ctx, ul, client.MatchingFields{constants.PropagateKey: constants.PropagateUpdate}, client.InNamespace(parent)); err != nil {
		return fmt.Errorf("failed to list %s in %s with propagate=update: %w", gvkStr, parent, err)
	}
	presNames := make(map[string]bool)
	for i := range ul.Items {
		pres := &ul.Items[i]
		presNames[pres.GetName()] = true
		if err := r.propagateUpdate(ctx, pres, ns); err != nil {
			return fmt.Errorf("failed to propagate resource %s/%s of %s with propagate=update: %w", ns, pres.GetName(), gvkStr, err)
		}
	}

	ul2 := l.DeepCopy()
	if err := r.List(ctx, ul2, client.MatchingFields{constants.PropagateKey: constants.PropagateUpdate}, client.InNamespace(ns)); err != nil {
		return fmt.Errorf("failed to list %s in %s with propagate=update: %w", gvkStr, ns, err)
	}
	for i := range ul2.Items {
		cres := &ul2.Items[i]
		from := cres.GetAnnotations()[constants.AnnFrom]
		if from == "" {
			// don't delete origins
			continue
		}

		if from == parent && presNames[cres.GetName()] {
			continue
		}
		if err := r.Delete(ctx, cres); err != nil {
			return fmt.Errorf("failed to delete stale resource %s/%s of %s: %w", ns, cres.GetName(), gvkStr, err)
		}
		logger.Info("deleted a resource", "namespace", cres.GetNamespace(), "name", cres.GetName(), "gvk", gvkStr)
	}

	return nil
}

func (r *NamespaceReconciler) propagateCreate(ctx context.Context, res *unstructured.Unstructured, ns string) error {
	gvk := res.GroupVersionKind()

	c := &unstructured.Unstructured{}
	c.SetGroupVersionKind(gvk)
	err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: res.GetName()}, c)
	if err == nil {
		return nil
	}
	if !apierrors.IsNotFound(err) {
		return err
	}

	if err := r.Create(ctx, cloneResource(res, ns)); err != nil {
		return err
	}

	logger := log.FromContext(ctx)
	logger.Info("created a resource", "namespace", ns, "name", res.GetName(), "gvk", gvk.String())
	return nil
}

func (r *NamespaceReconciler) propagateUpdate(ctx context.Context, res *unstructured.Unstructured, ns string) error {
	logger := log.FromContext(ctx)
	gvk := res.GroupVersionKind()

	c := &unstructured.Unstructured{}
	c.SetGroupVersionKind(gvk)
	err := r.Get(ctx, client.ObjectKey{Namespace: ns, Name: res.GetName()}, c)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		if err := r.Create(ctx, cloneResource(res, ns)); err != nil {
			return err
		}
		logger.Info("created a resource", "namespace", ns, "name", res.GetName(), "gvk", gvk.String())
		return nil
	}

	c2 := cloneResource(res, ns)
	if equality.Semantic.Equalities.DeepDerivative(c2, c) {
		return nil
	}

	c2.SetResourceVersion(c.GetResourceVersion())
	if err := r.Update(ctx, c2); err != nil {
		return err
	}

	logger.Info("updated a resource", "namespace", ns, "name", res.GetName(), "gvk", gvk.String())
	return nil
}

func (r *NamespaceReconciler) deleteResource(ctx context.Context, res *unstructured.Unstructured, ns string) error {
	logger := log.FromContext(ctx)

	gvk := res.GroupVersionKind()
	gvkStr := gvk.String()
	gvk.Kind = gvk.Kind + "List"
	l := &unstructured.UnstructuredList{}
	l.SetGroupVersionKind(gvk)

	if err := r.List(ctx, l, client.MatchingFields{constants.PropagateKey: constants.PropagateUpdate}, client.InNamespace(ns)); err != nil {
		return fmt.Errorf("failed to list %s in %s: %w", gvkStr, ns, err)
	}
	for i := range l.Items {
		obj := &l.Items[i]
		if err := r.Delete(ctx, obj); err != nil {
			return fmt.Errorf("failed to delete %s/%s of %s: %w", ns, obj.GetName(), gvkStr, err)
		}
		logger.Info("deleted a resource", "namespace", ns, "name", obj.GetName(), "gvk", gvkStr)
	}
	return nil
}

func (r *NamespaceReconciler) deletePropagatedResources(ctx context.Context, ns *corev1.Namespace) error {
	for _, res := range r.Watched {
		if err := r.deleteResource(ctx, res, ns.Name); err != nil {
			return err
		}
	}
	return nil
}

func (r *NamespaceReconciler) reconcileSubNamespace(ctx context.Context, ns *corev1.Namespace, parent string) error {
	parentNS := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: parent}, parentNS); err != nil {
		return fmt.Errorf("failed to get parent namespace %s: %w", parent, err)
	}
	if err := r.propagateMeta(ctx, ns, parentNS); err != nil {
		return err
	}

	children := &corev1.NamespaceList{}
	if err := r.List(ctx, children, client.MatchingFields{constants.NamespaceParentKey: ns.Name}); err != nil {
		return fmt.Errorf("failed to list the children: %w", err)
	}
	for i := range children.Items {
		child := &children.Items[i]
		if err := r.propagateMeta(ctx, child, ns); err != nil {
			return err
		}
	}

	for _, res := range r.Watched {
		if err := r.propagateResource(ctx, res, parent, ns.Name); err != nil {
			return err
		}
	}

	return nil
}

func (r *NamespaceReconciler) reconcileRootNamespace(ctx context.Context, ns *corev1.Namespace) error {
	subs := &corev1.NamespaceList{}
	if err := r.List(ctx, subs, client.MatchingFields{constants.NamespaceParentKey: ns.Name}); err != nil {
		return fmt.Errorf("failed to list sub namespaces: %w", err)
	}

	for i := range subs.Items {
		sub := &subs.Items[i]
		if err := r.propagateMeta(ctx, sub, ns); err != nil {
			return err
		}
	}
	return nil
}

func (r *NamespaceReconciler) reconcileInstanceNamespace(ctx context.Context, ns *corev1.Namespace, tmpl string) error {
	tmplNS := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: tmpl}, tmplNS); err != nil {
		return fmt.Errorf("failed to get template namespace %s: %w", tmpl, err)
	}

	if err := r.propagateMeta(ctx, ns, tmplNS); err != nil {
		return err
	}

	for _, res := range r.Watched {
		if err := r.propagateResource(ctx, res, tmpl, ns.Name); err != nil {
			return err
		}
	}

	return nil
}

func (r *NamespaceReconciler) reconcileTemplateNamespace(ctx context.Context, ns *corev1.Namespace) error {
	instances := &corev1.NamespaceList{}
	if err := r.List(ctx, instances, client.MatchingFields{constants.NamespaceTemplateKey: ns.Name}); err != nil {
		return fmt.Errorf("failed to list instance namespaces: %w", err)
	}

	for i := range instances.Items {
		instance := &instances.Items[i]
		if err := r.propagateMeta(ctx, instance, ns); err != nil {
			return err
		}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		Complete(r)
}
