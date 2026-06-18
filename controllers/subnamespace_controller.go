package controllers

import (
	"context"
	"fmt"
	"time"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	accuratev2ac "github.com/cybozu-go/accurate/internal/applyconfigurations/accurate/v2"
	"github.com/cybozu-go/accurate/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/json"
	metav1ac "k8s.io/client-go/applyconfigurations/meta/v1"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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

	sn := &accuratev2.SubNamespace{}
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

func (r *SubNamespaceReconciler) finalize(ctx context.Context, sn *accuratev2.SubNamespace) error {
	logger := log.FromContext(ctx)

	ns := &corev1.Namespace{}
	if err := r.Get(ctx, types.NamespacedName{Name: sn.Name}, ns); err != nil {
		if !apierrors.IsNotFound(err) {
			return err
		}
		return r.removeFinalizer(ctx, sn)
	}

	if ns.DeletionTimestamp != nil {
		return r.removeFinalizer(ctx, sn)
	}

	if parent := ns.Labels[constants.LabelParent]; parent != sn.Namespace {
		logger.Info("finalization: ignored non-child namespace", "parent", parent)
		return r.removeFinalizer(ctx, sn)
	}

	if err := r.Delete(ctx, ns); err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete namespace %s: %w", ns.Name, err)
		}
	} else {
		logger.Info("deleted namespace", "name", ns.Name)
	}

	return r.removeFinalizer(ctx, sn)
}

func (r *SubNamespaceReconciler) removeFinalizer(ctx context.Context, sn *accuratev2.SubNamespace) error {
	if !controllerutil.ContainsFinalizer(sn, constants.Finalizer) {
		return nil
	}

	sn = sn.DeepCopy()
	controllerutil.RemoveFinalizer(sn, constants.Finalizer)

	// We'll use a JSON Merge Patch here to avoid removal of finalizers added by other controllers
	patch := map[string]any{
		"metadata": map[string]any{
			"finalizers": sn.GetFinalizers(),
		},
	}
	patchBytes, err := json.Marshal(patch)
	if err != nil {
		return err
	}

	return client.IgnoreNotFound(r.Patch(ctx, sn, client.RawPatch(types.MergePatchType, patchBytes)))
}

func (r *SubNamespaceReconciler) reconcileNS(ctx context.Context, sn *accuratev2.SubNamespace) error {
	logger := log.FromContext(ctx)

	// should create a namespace even when spec.parent differs from the namespace
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
			return fmt.Errorf("failed to create namespace %s: %w", ns.Name, err)
		}
		logger.Info("created a sub namespace", "name", sn.Name)
	}

	// Move reconciliation is driven by the source SubNamespace.
	// When the target SubNamespace becomes move-accepted, the source SubNamespace
	// is enqueued and completes the move.
	if sn.IsMoveRequested() {
		return r.reconcileMove(ctx, sn, ns)
	}

	ac := subNamespaceStatusApplyConfig(sn)

	if ns.Labels[constants.LabelParent] != sn.Namespace {
		logger.Info("a conflicting namespace already exists")
		withStalledCondition(
			ac, sn,
			accuratev2.SubNamespaceReasonConflict,
			"Conflicting namespace already exists",
		)
	}

	return r.Status().Apply(ctx, ac, fieldOwner, client.ForceOwnership)
}

// reconcileMove reconciles a move request on the source SubNamespace.
func (r *SubNamespaceReconciler) reconcileMove(ctx context.Context, sn *accuratev2.SubNamespace, childNS *corev1.Namespace) error {
	logger := log.FromContext(ctx)

	targetParent := sn.MoveTargetParent()

	target := &accuratev2.SubNamespace{}
	targetKey := types.NamespacedName{
		Namespace: targetParent,
		Name:      sn.Name,
	}

	if err := r.Get(ctx, targetKey, target); err != nil {
		if apierrors.IsNotFound(err) {
			ac := subNamespaceStatusApplyConfig(sn)
			withMoveStalledCondition(
				ac, sn,
				accuratev2.SubNamespaceReasonMoveTargetNotFound,
				fmt.Sprintf("Target SubNamespace %s/%s does not exist", targetParent, sn.Name),
			)
			return r.Status().Apply(ctx, ac, fieldOwner, client.ForceOwnership)
		}
		return err
	}

	if !target.AcceptsMoveFrom(sn.Namespace) {
		ac := subNamespaceStatusApplyConfig(sn)
		withMoveStalledCondition(
			ac, sn,
			accuratev2.SubNamespaceReasonMoveNotAccepted,
			fmt.Sprintf(
				"Target SubNamespace %q/%q has not accepted a move from %q",
				targetParent,
				sn.Name,
				sn.Namespace,
			),
		)
		return r.Status().Apply(ctx, ac, fieldOwner, client.ForceOwnership)
	}

	currentParent := childNS.Labels[constants.LabelParent]
	if sn.Namespace != currentParent {
		ac := subNamespaceStatusApplyConfig(sn)
		withStalledCondition(
			ac, sn,
			accuratev2.SubNamespaceReasonConflict,
			"Conflicting namespace already exists",
		)

		if targetParent != currentParent {
			withMoveStalledCondition(
				ac, sn,
				accuratev2.SubNamespaceReasonMoveNotAccepted,
				fmt.Sprintf(
					"Target SubNamespace %q/%q has not accepted a move from %q",
					targetParent,
					sn.Name,
					sn.Namespace,
				),
			)
		}

		return r.Status().Apply(ctx, ac, fieldOwner, client.ForceOwnership)
	}

	base := childNS.DeepCopy()
	if childNS.Labels == nil {
		childNS.Labels = map[string]string{}
	}
	childNS.Labels[constants.LabelParent] = targetParent

	if err := r.Patch(ctx, childNS, client.MergeFrom(base)); err != nil {
		return fmt.Errorf("failed to update parent label of namespace %s: %w", childNS.Name, err)
	}

	logger.Info(
		"updated sub namespace parent label",
		"name", sn.Name,
		"sourceParent", sn.Namespace,
		"targetParent", targetParent,
	)

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SubNamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Move reconciliation is performed by the source SubNamespace,
	// so changes to a move-accepting target must trigger reconciliation of matching sources.
	moveTargetHandler := func(ctx context.Context, o client.Object) (requests []reconcile.Request) {
		target, ok := o.(*accuratev2.SubNamespace)
		if !ok {
			return nil
		}

		if !target.IsAcceptingMove() {
			return nil
		}

		snList := &accuratev2.SubNamespaceList{}
		if err := r.List(ctx, snList, client.MatchingFields{
			constants.SubNamespaceNameKey: target.Name,
		}); err != nil {
			logger := log.FromContext(ctx)
			logger.Error(err, "failed to list subnamespaces")
			return nil
		}

		for _, sn := range snList.Items {
			if !sn.IsMoveRequested() {
				continue
			}

			if !target.AcceptsMoveFrom(sn.Namespace) {
				continue
			}

			requests = append(requests, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: sn.Namespace,
					Name:      sn.Name,
				},
			})
		}

		return requests
	}

	nsHandler := func(ctx context.Context, o client.Object) (requests []reconcile.Request) {
		// Reconcile all SubNamespaces with the same name when the Namespace changes,
		// such as parent label updates or deletion.
		snList := &accuratev2.SubNamespaceList{}
		err := r.List(ctx, snList, client.MatchingFields{constants.SubNamespaceNameKey: o.GetName()})
		if err != nil {
			logger := log.FromContext(ctx)
			logger.Error(err, "failed to list subnamespaces")
			return
		}
		for _, sn := range snList.Items {
			requests = append(requests, reconcile.Request{NamespacedName: types.NamespacedName{
				Namespace: sn.Namespace,
				Name:      sn.Name,
			}})
		}

		return
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&accuratev2.SubNamespace{}).
		// Requeue the source SubNamespace when the target SubNamespace becomes move-accepted.
		Watches(
			&accuratev2.SubNamespace{},
			handler.EnqueueRequestsFromMapFunc(moveTargetHandler),
			builder.WithPredicates(predicate.Funcs{
				CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
					sn, ok := e.Object.(*accuratev2.SubNamespace)
					return ok && sn.IsAcceptingMove()
				},
				UpdateFunc: func(e event.TypedUpdateEvent[client.Object]) bool {
					oldSN, oldOK := e.ObjectOld.(*accuratev2.SubNamespace)
					newSN, newOK := e.ObjectNew.(*accuratev2.SubNamespace)
					if !oldOK || !newOK {
						return false
					}

					if !newSN.IsAcceptingMove() {
						return false
					}

					return oldSN.MoveSourceParent() != newSN.MoveSourceParent()
				},
				DeleteFunc: func(e event.TypedDeleteEvent[client.Object]) bool {
					return false
				},
			}),
		).
		Watches(&corev1.Namespace{}, handler.EnqueueRequestsFromMapFunc(nsHandler), builder.WithPredicates(predicate.Funcs{
			CreateFunc: func(e event.TypedCreateEvent[client.Object]) bool {
				return false
			},
		})).
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

func subNamespaceStatusApplyConfig(sn *accuratev2.SubNamespace) *accuratev2ac.SubNamespaceApplyConfiguration {
	return accuratev2ac.SubNamespace(sn.Name, sn.Namespace).
		WithStatus(
			accuratev2ac.SubNamespaceStatus().
				WithObservedGeneration(sn.Generation),
		)
}

func withMoveStalledCondition(
	ac *accuratev2ac.SubNamespaceApplyConfiguration,
	sn *accuratev2.SubNamespace,
	reason string,
	message string,
) {
	ac.Status.WithConditions(
		conditionPatch(
			sn.Status.Conditions,
			metav1ac.Condition().
				WithType(accuratev2.SubNamespaceTypeMoveStalled).
				WithStatus(metav1.ConditionTrue).
				WithObservedGeneration(sn.Generation).
				WithReason(reason).
				WithMessage(message),
		),
	)
}

func withStalledCondition(
	ac *accuratev2ac.SubNamespaceApplyConfiguration,
	sn *accuratev2.SubNamespace,
	reason string,
	message string,
) {
	ac.Status.WithConditions(
		conditionPatch(
			sn.Status.Conditions,
			metav1ac.Condition().
				WithType(string(kstatus.ConditionStalled)).
				WithStatus(metav1.ConditionTrue).
				WithObservedGeneration(sn.Generation).
				WithReason(reason).
				WithMessage(message),
		),
	)
}
