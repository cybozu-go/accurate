package controllers

import (
	"context"
	"fmt"
	"strings"

	"github.com/cybozu-go/innu/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

const notGenerated = "false"

func cloneResource(res *unstructured.Unstructured, ns string) *unstructured.Unstructured {
	c := res.DeepCopy()
	delete(c.Object, "metadata")
	delete(c.Object, "status")
	c.SetNamespace(ns)
	c.SetName(res.GetName())
	labels := make(map[string]string)
	for k, v := range res.GetLabels() {
		if strings.Contains(k, "kubernetes.io/") {
			continue
		}
		labels[k] = v
	}
	labels[constants.LabelCreatedBy] = constants.CreatedBy
	c.SetLabels(labels)
	annotations := make(map[string]string)
	for k, v := range res.GetAnnotations() {
		if strings.Contains(k, "kubernetes.io/") {
			continue
		}
		annotations[k] = v
	}
	annotations[constants.AnnFrom] = res.GetNamespace()
	c.SetAnnotations(annotations)

	return c
}

// PropagateController propagates objects of a namespace-scoped resource.
type PropagateController struct {
	client.Client
	reader client.Reader
	res    *unstructured.Unstructured
}

// NewPropagateController creates a new PropagateController.
// The GroupVersionKind of `res` must be set.
func NewPropagateController(res *unstructured.Unstructured) *PropagateController {
	if res.GetKind() == "" {
		panic("no group version kind")
	}
	return &PropagateController{
		res: res.DeepCopy(),
	}
}

// Reconcile implements reconcile.Reconciler interface.
func (r *PropagateController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(5).Info("reconciling")

	obj := r.res.DeepCopy()
	if err := r.Get(ctx, req.NamespacedName, obj); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
		if err := r.handleDelete(ctx, req); err != nil {
			logger.Error(err, "failed to handle deleted object")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if obj.GetDeletionTimestamp() != nil {
		return ctrl.Result{}, nil
	}

	ann := obj.GetAnnotations()
	if from := ann[constants.AnnFrom]; from != "" {
		p := r.res.DeepCopy()
		if err := r.Get(ctx, client.ObjectKey{Namespace: from, Name: req.Name}, p); err != nil {
			if !apierrors.IsNotFound(err) {
				return ctrl.Result{}, fmt.Errorf("failed to lookup the parent resource in %s: %w", from, err)
			}

			if ann[constants.AnnPropagate] == constants.PropagateUpdate {
				if err := r.Delete(ctx, obj); err != nil {
					logger.Error(err, "failed to delete")
					return ctrl.Result{}, err
				}
				logger.Info("deleted")
				return ctrl.Result{}, nil
			}
		} else {
			if p.GetAnnotations()[constants.AnnPropagate] == constants.PropagateUpdate {
				if err := r.propagateUpdate(ctx, obj, p); err != nil {
					logger.Error(err, "failed to propagate an object", "mode", "update", "parent", "exist")
					return ctrl.Result{}, err
				}
				return ctrl.Result{}, nil
			}
		}
	}

	switch ann[constants.AnnPropagate] {
	case constants.PropagateCreate:
		if err := r.propagateCreate(ctx, obj); err != nil {
			logger.Error(err, "failed to propagate an object", "mode", "create")
			return ctrl.Result{}, err
		}
	case constants.PropagateUpdate:
		if err := r.propagateUpdate(ctx, obj, nil); err != nil {
			logger.Error(err, "failed to propagate an object", "mode", "update", "parent", "none")
			return ctrl.Result{}, err
		}
	case "":
		if ann[constants.AnnGenerated] != notGenerated {
			if err := r.checkController(ctx, obj); err != nil {
				logger.Error(err, "failed to check the controller reference")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}

func (r *PropagateController) getChildren(ctx context.Context, name string) (*corev1.NamespaceList, error) {
	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: name}, ns); err != nil {
		return nil, fmt.Errorf("failed to get namespace %s: %w", name, err)
	}

	children := &corev1.NamespaceList{}
	mf := client.MatchingFields{constants.NamespaceTemplateKey: name}
	if ns.Labels[constants.LabelRoot] == "true" || ns.Labels[constants.LabelParent] != "" {
		mf = client.MatchingFields{constants.NamespaceParentKey: name}
	}
	if err := r.List(ctx, children, mf); err != nil {
		return nil, fmt.Errorf("failed to list children namespaces: %w", err)
	}
	return children, nil
}

func (r *PropagateController) handleDelete(ctx context.Context, req ctrl.Request) error {
	logger := log.FromContext(ctx)

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: req.Namespace}, ns); err != nil {
		return fmt.Errorf("failed to get namespace %s: %w", req.Namespace, err)
	}

	// re-create it if there is a parent resource
	p, ok := ns.Labels[constants.LabelParent]
	if !ok {
		p = ns.Labels[constants.LabelTemplate]
	}
	if p != "" {
		parent := &corev1.Namespace{}
		if err := r.Get(ctx, client.ObjectKey{Name: p}, parent); err != nil {
			kind := "parent"
			if !ok {
				kind = "template"
			}
			return fmt.Errorf("failed to get %s namespace %s: %w", kind, p, err)
		}

		obj := r.res.DeepCopy()
		if err := r.Get(ctx, client.ObjectKey{Namespace: p, Name: req.Name}, obj); err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get %s/%s: %w", p, req.Name, err)
			}
		} else {
			switch obj.GetAnnotations()[constants.AnnPropagate] {
			case constants.PropagateCreate, constants.PropagateUpdate:
				if err := r.Create(ctx, cloneResource(obj, req.Namespace)); err != nil {
					return fmt.Errorf("failed to re-create %s/%s: %w", req.Namespace, req.Name, err)
				}
				logger.Info("re-created", "from", fmt.Sprintf("%s/%s", p, req.Name))
				return nil
			}
		}
	}

	// delete propagated resources in child namespaces
	children, err := r.getChildren(ctx, req.Namespace)
	if err != nil {
		return err
	}
	for _, child := range children.Items {
		obj := r.res.DeepCopy()
		if err := r.Get(ctx, client.ObjectKey{Namespace: child.Name, Name: req.Name}, obj); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("failed to look up %s/%s: %w", child.Name, req.Name, err)
		}

		if obj.GetAnnotations()[constants.AnnPropagate] != constants.PropagateUpdate {
			continue
		}

		if err := r.Delete(ctx, obj); err != nil {
			return fmt.Errorf("failed to cascade delete %s/%s: %w", child.Name, req.Name, err)
		}
		logger.Info("deleted a child resource", "subnamespace", child.Name)
	}

	return nil
}

func (r *PropagateController) propagateCreate(ctx context.Context, obj *unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	children, err := r.getChildren(ctx, obj.GetNamespace())
	if err != nil {
		return err
	}

	name := obj.GetName()
	for _, child := range children.Items {
		cres := r.res.DeepCopy()
		err := r.Get(ctx, client.ObjectKey{Namespace: child.Name, Name: name}, cres)
		if err == nil {
			continue
		}
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to look up %s/%s: %w", child.Name, name, err)
		}

		if err := r.Create(ctx, cloneResource(obj, child.Name)); err != nil {
			return fmt.Errorf("failed to create %s/%s: %w", child.Name, name, err)
		}

		logger.Info("created a child resource", "subnamespace", child.Name)
	}

	return nil
}

func (r *PropagateController) propagateUpdate(ctx context.Context, obj, parent *unstructured.Unstructured) error {
	logger := log.FromContext(ctx)
	name := obj.GetName()

	if parent != nil {
		clone := cloneResource(parent, obj.GetNamespace())
		if !equality.Semantic.DeepDerivative(clone, obj) {
			clone.SetResourceVersion(obj.GetResourceVersion())
			if err := r.Update(ctx, clone); err != nil {
				return fmt.Errorf("failed to update: %w", err)
			}
			logger.Info("updated", "from", parent.GetNamespace())
			return nil
		}
	}

	// propagate to child namespaces, if any.
	children, err := r.getChildren(ctx, obj.GetNamespace())
	if err != nil {
		return err
	}

	for _, child := range children.Items {
		cres := r.res.DeepCopy()
		err := r.Get(ctx, client.ObjectKey{Namespace: child.Name, Name: name}, cres)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to lookup %s/%s: %w", child.Name, name, err)
			}

			clone := cloneResource(obj, child.Name)
			if err := r.Create(ctx, clone); err != nil {
				return fmt.Errorf("failed to create %s/%s: %w", child.Name, name, err)
			}

			logger.Info("created a child resource", "subnamespace", child.Name)
			continue
		}

		clone := cloneResource(obj, child.Name)
		if equality.Semantic.DeepDerivative(clone, cres) {
			continue
		}

		clone.SetResourceVersion(cres.GetResourceVersion())
		if err := r.Update(ctx, clone); err != nil {
			return fmt.Errorf("failed to update %s/%s: %w", child.Name, name, err)
		}

		logger.Info("updated a child resource", "subnamespace", child.Name)
	}

	return nil
}

func (r *PropagateController) checkController(ctx context.Context, obj *unstructured.Unstructured) error {
	cref := metav1.GetControllerOfNoCopy(obj)
	if cref == nil {
		return nil
	}

	logger := log.FromContext(ctx)
	owner := &unstructured.Unstructured{}
	owner.SetGroupVersionKind(schema.FromAPIVersionAndKind(cref.APIVersion, cref.Kind))
	if err := r.reader.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: cref.Name}, owner); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("the controller object is not found", "gvk", owner.GroupVersionKind().String(), "owner", cref.Name)
			return nil
		}
		return err
	}

	patched := obj.DeepCopy()
	ann := patched.GetAnnotations()
	if ann == nil {
		ann = make(map[string]string)
	}
	mode, ok := owner.GetAnnotations()[constants.AnnPropagateGenerated]
	if !ok {
		ann[constants.AnnGenerated] = notGenerated
	} else {
		ann[constants.AnnPropagate] = mode
	}
	patched.SetAnnotations(ann)

	if err := r.Patch(ctx, patched, client.MergeFrom(obj)); err != nil {
		return fmt.Errorf("failed to add %s annotation: %w", constants.AnnPropagateGenerated, err)
	}

	logger.Info("annotated to store the result of checking the owner")
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PropagateController) SetupWithManager(mgr ctrl.Manager) error {
	pred := func(obj client.Object) bool {
		ann := obj.GetAnnotations()
		if _, ok := ann[constants.AnnFrom]; ok {
			return true
		}
		if _, ok := ann[constants.AnnPropagate]; ok {
			return true
		}
		if ann[constants.AnnGenerated] == notGenerated {
			return false
		}
		if metav1.GetControllerOfNoCopy(obj) != nil {
			return true
		}
		return false
	}

	r.Client = mgr.GetClient()
	r.reader = mgr.GetAPIReader()

	return ctrl.NewControllerManagedBy(mgr).
		For(r.res).
		WithEventFilter(predicate.Funcs{
			CreateFunc: func(e event.CreateEvent) bool { return pred(e.Object) },
			UpdateFunc: func(e event.UpdateEvent) bool { return pred(e.ObjectOld) || pred(e.ObjectNew) },
			DeleteFunc: func(e event.DeleteEvent) bool { return pred(e.Object) },
		}).
		Complete(r)
}
