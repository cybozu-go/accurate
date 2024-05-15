package controllers

import (
	"context"
	"fmt"
	"time"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	accuratev2alpha1ac "github.com/cybozu-go/accurate/internal/applyconfigurations/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"
	"k8s.io/client-go/util/workqueue"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// SubNamespaceReconciler reconciles a SubNamespace object
type SubNamespaceReconciler struct {
	client.Client
}

//+kubebuilder:rbac:groups=accurate.cybozu.com,resources=subnamespaces,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=accurate.cybozu.com,resources=subnamespaces/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=accurate.cybozu.com,resources=subnamespaces/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;patch;delete

// Reconcile implements reconcile.Reconciler interface.
func (r *SubNamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	sn := &accuratev2alpha1.SubNamespace{}
	if err := r.Get(ctx, req.NamespacedName, sn); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if sn.DeletionTimestamp != nil {
		logger.Info("starting finalization")
		if err := r.finalize(ctx, sn); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to finalize: %w", err)
		}
		logger.Info("finished finalization")
		return ctrl.Result{}, nil
	}

	if err := r.reconcileNS(ctx, sn); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile: %w", err)
	}

	return ctrl.Result{}, nil
}

func (r *SubNamespaceReconciler) finalize(ctx context.Context, sn *accuratev2alpha1.SubNamespace) error {
	if !controllerutil.ContainsFinalizer(sn, constants.Finalizer) {
		return nil
	}

	logger := log.FromContext(ctx)

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: sn.Name}, ns); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		goto DELETE
	}

	if ns.DeletionTimestamp != nil {
		goto DELETE
	}

	if parent := ns.Labels[constants.LabelParent]; parent != sn.Namespace {
		logger.Info("finalization: ignored non-child namespace", "parent", parent)
		goto DELETE
	}

	if err := r.Delete(ctx, ns); err != nil {
		return fmt.Errorf("failed to delete namespace %s: %w", sn.Name, err)
	}

	logger.Info("deleted namespace", "name", sn.Name)

DELETE:
	orig := sn.DeepCopy()
	controllerutil.RemoveFinalizer(sn, constants.Finalizer)
	return r.Patch(ctx, sn, client.MergeFrom(orig))
}

func (r *SubNamespaceReconciler) reconcileNS(ctx context.Context, sn *accuratev2alpha1.SubNamespace) error {
	logger := log.FromContext(ctx)

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, client.ObjectKey{Name: sn.Name}, ns); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}

		ns = &corev1.Namespace{}
		ns.Name = sn.Name
		ns.Labels = map[string]string{
			constants.LabelCreatedBy: constants.CreatedBy,
			constants.LabelParent:    sn.Namespace,
		}
		if err := r.Create(ctx, ns); err != nil {
			return err
		}
		logger.Info("created a sub namespace", "name", sn.Name)
	}

	ac := accuratev2alpha1ac.SubNamespace(sn.Name, sn.Namespace).
		WithStatus(
			accuratev2alpha1ac.SubNamespaceStatus().
				WithObservedGeneration(sn.Generation),
		)

	if ns.Labels[constants.LabelParent] != sn.Namespace {
		logger.Info("a conflicting namespace already exists")
		ac.Status.WithConditions(
			conditionPatch(sn.Status.Conditions,
				metav1ac.Condition().
					WithType(string(kstatus.ConditionStalled)).
					WithStatus(metav1.ConditionTrue).
					WithObservedGeneration(sn.Generation).
					WithReason(accuratev2alpha1.SubNamespaceConflict).
					WithMessage("Conflicting namespace already exists"),
			),
		)
	}

	// TODO: upgrade managed fields to SSA when https://github.com/kubernetes/kubernetes/pull/123484 is released

	sn, p, err := newSubNamespacePatch(ac)
	if err != nil {
		return err
	}
	return r.Status().Patch(ctx, sn, p, fieldOwner, client.ForceOwnership)
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	nsHandler := func(o client.Object, q workqueue.RateLimitingInterface) {
		parent := o.GetLabels()[constants.LabelParent]
		if parent == "" {
			return
		}
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Namespace: parent,
			Name:      o.GetName(),
		}})
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&accuratev2alpha1.SubNamespace{}).
		Watches(&corev1.Namespace{}, handler.Funcs{
			UpdateFunc: func(ctx context.Context, ev event.UpdateEvent, q workqueue.RateLimitingInterface) {
				if ev.ObjectNew.GetDeletionTimestamp() != nil {
					return
				}
				nsHandler(ev.ObjectOld, q)
			},
			DeleteFunc: func(ctx context.Context, ev event.DeleteEvent, q workqueue.RateLimitingInterface) {
				nsHandler(ev.Object, q)
			},
		}).
		Complete(r)
}

func conditionPatch(existingConditions []metav1.Condition, condition *metav1ac.ConditionApplyConfiguration) *metav1ac.ConditionApplyConfiguration {
	if condition.LastTransitionTime.IsZero() {
		existingCondition := meta.FindStatusCondition(existingConditions, *condition.Type)
		if existingCondition != nil && existingCondition.Status == *condition.Status {
			condition.WithLastTransitionTime(existingCondition.LastTransitionTime)
		} else {
			condition.WithLastTransitionTime(metav1.NewTime(time.Now()))
		}
	}

	return condition
}
