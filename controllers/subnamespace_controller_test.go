package controllers

import (
	"context"
	"time"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/cybozu-go/accurate/pkg/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	gomegaTypes "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kstatus "sigs.k8s.io/cli-utils/pkg/kstatus/status"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("SubNamespace controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(k8sCfg, ctrl.Options{
			Scheme:         scheme,
			LeaderElection: false,
			Metrics:        server.Options{BindAddress: "0"},
			Controller: config.Controller{
				SkipNameValidation: new(true),
			},
		})
		Expect(err).ToNot(HaveOccurred())

		snr := &SubNamespaceReconciler{
			Client: mgr.GetClient(),
		}
		err = snr.SetupWithManager(mgr)
		Expect(err).ToNot(HaveOccurred())

		err = indexing.SetupIndexForSubNamespace(ctx, mgr)
		Expect(err).NotTo(HaveOccurred())

		ctx, cancel := context.WithCancel(ctx)
		stopFunc = cancel
		go func() {
			err := mgr.Start(ctx)
			if err != nil {
				panic(err)
			}
		}()
		time.Sleep(100 * time.Millisecond)
	})

	AfterEach(func() {
		stopFunc()
		time.Sleep(100 * time.Millisecond)
	})

	It("should create and delete sub-namespaces", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test1"
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sn := &accuratev2.SubNamespace{}
		sn.Namespace = "test1"
		sn.Name = "test1-sub1"
		sn.Finalizers = []string{constants.Finalizer}
		Expect(k8sClient.Create(ctx, sn)).To(Succeed())

		sub1 := &corev1.Namespace{}
		sub1.Name = "test1-sub1"
		Eventually(komega.Get(sub1)).Should(Succeed())

		Expect(sub1.Labels).To(HaveKeyWithValue(constants.LabelCreatedBy, "accurate"))
		Expect(sub1.Labels).To(HaveKeyWithValue(constants.LabelParent, "test1"))
		Eventually(komega.Object(sn)).Should(HaveField("Status.ObservedGeneration", BeNumerically(">", 0)))
		Expect(sn.Status.Conditions).To(BeEmpty())

		Expect(k8sClient.Delete(ctx, sn)).To(Succeed())

		Eventually(komega.Object(sub1)).Should(HaveField("DeletionTimestamp", Not(BeNil())))
	})

	It("should detect conflicts", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test2"
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		ns2 := &corev1.Namespace{}
		ns2.Name = "test2-sub1"
		Expect(k8sClient.Create(ctx, ns2)).To(Succeed())

		sn := &accuratev2.SubNamespace{}
		sn.Namespace = "test2"
		sn.Name = "test2-sub1"
		Expect(k8sClient.Create(ctx, sn)).To(Succeed())

		Eventually(komega.Object(sn)).Should(HaveField("Status.ObservedGeneration", BeNumerically(">", 0)))
		Expect(sn.Status.Conditions).To(HaveLen(1))
		Expect(sn.Status.Conditions[0].Reason).To(Equal(accuratev2.SubNamespaceReasonConflict))

		// It's tempting to test if a conflict can be resolved by deleting the conflicting namespace,
		// but this is currently not possible because EnvTest does not support namespace deletion.
		// See https://github.com/kubernetes-sigs/controller-runtime/issues/880 for details.
		// This feature should be tested in e2e-tests.
	})

	It("should not delete a conflicting sub-namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test3"
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sn := &accuratev2.SubNamespace{}
		sn.Namespace = "test3"
		sn.Name = "test3-sub1"
		sn.Finalizers = []string{constants.Finalizer}
		Expect(k8sClient.Create(ctx, sn)).To(Succeed())

		sub1 := &corev1.Namespace{}
		sub1.Name = "test3-sub1"
		Eventually(komega.Get(sub1)).Should(Succeed())

		Expect(komega.Update(sub1, func() {
			sub1.Labels[constants.LabelParent] = "foo"
		})()).To(Succeed())

		Eventually(komega.Object(sn)).Should(HaveField("Status.Conditions", HaveLen(1)))
		Expect(sn.Status.Conditions[0].Reason).To(Equal(accuratev2.SubNamespaceReasonConflict))

		Expect(k8sClient.Delete(ctx, sn)).To(Succeed())

		Consistently(komega.Object(sub1)).Should(HaveField("DeletionTimestamp", BeNil()))
	})

	It("should re-create a sub-namespace if it is deleted", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test4"
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sn := &accuratev2.SubNamespace{}
		sn.Namespace = "test4"
		sn.Name = "test4-sub1"
		Expect(k8sClient.Create(ctx, sn)).To(Succeed())

		sub1 := &corev1.Namespace{}
		sub1.Name = "test4-sub1"
		Eventually(komega.Get(sub1)).Should(Succeed())

		Expect(k8sClient.Delete(ctx, sub1)).To(Succeed())

		uid := sub1.UID
		sub1 = &corev1.Namespace{}
		sub1.Name = "test4-sub1"
		cs, err := kubernetes.NewForConfig(k8sCfg)
		Expect(err).NotTo(HaveOccurred())
		_, err = cs.CoreV1().Namespaces().Finalize(ctx, sub1, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(komega.Object(sub1)).Should(HaveField("UID", Not(Equal(uid))))
	})

	It("should move a sub-namespace when the target SubNamespace accepts the source parent", func() {
		sourceParent := "test5-old"
		targetParent := "test5-new"
		childName := "test5-sub1"

		createNamespace(ctx, sourceParent)
		createNamespace(ctx, targetParent)

		sourceSN := createSubNamespace(ctx, sourceParent, childName, "", "")
		expectNamespaceParent(ctx, childName, sourceParent)

		Eventually(komega.Object(sourceSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", notHaveConditionType(accuratev2.SubNamespaceTypeMoveStalled)),
			HaveField("Status.Conditions", notHaveConditionType(string(kstatus.ConditionStalled))),
		))

		targetSN := createSubNamespace(ctx, targetParent, childName, sourceParent, "")

		Eventually(komega.Object(targetSN)).Should(
			HaveField("Status.Conditions",
				haveCondition(
					string(kstatus.ConditionStalled),
					metav1.ConditionTrue,
					accuratev2.SubNamespaceReasonConflict,
				),
			),
		)

		Expect(komega.Update(sourceSN, func() {
			sourceSN.Spec.Move = &accuratev2.SubNamespaceMoveSpec{
				TargetParent: targetParent,
			}
		})()).To(Succeed())

		expectNamespaceParent(ctx, childName, targetParent)

		Eventually(komega.Object(sourceSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", haveCondition(
				string(kstatus.ConditionStalled),
				metav1.ConditionTrue,
				accuratev2.SubNamespaceReasonConflict,
			)),
			HaveField("Status.Conditions", notHaveConditionType(accuratev2.SubNamespaceTypeMoveStalled))),
		)
	})

	It("should set MoveStalled=True when target SubNamespace does not exist", func() {
		sourceParent := "test6-old"
		targetParent := "test6-new"
		childName := "test6-sub1"

		createNamespace(ctx, sourceParent)
		createNamespace(ctx, targetParent)

		sourceSN := createSubNamespace(ctx, sourceParent, childName, "", "")
		expectNamespaceParent(ctx, childName, sourceParent)

		Expect(komega.Update(sourceSN, func() {
			sourceSN.Spec.Move = &accuratev2.SubNamespaceMoveSpec{
				TargetParent: targetParent,
			}
		})()).To(Succeed())

		Eventually(komega.Object(sourceSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", haveCondition(
				accuratev2.SubNamespaceTypeMoveStalled,
				metav1.ConditionTrue,
				accuratev2.SubNamespaceReasonMoveTargetNotFound,
			)),
			HaveField("Status.Conditions", notHaveConditionType(string(kstatus.ConditionStalled))),
		))

		expectNamespaceParent(ctx, childName, sourceParent)
	})

	It("should set MoveStalled=True when the target SubNamespace does not accept the source parent", func() {
		sourceParent := "test7-old"
		targetParent := "test7-new"
		otherParent := "test7-other"
		childName := "test7-sub1"

		createNamespace(ctx, sourceParent)
		createNamespace(ctx, targetParent)
		createNamespace(ctx, otherParent)

		sourceSN := createSubNamespace(ctx, sourceParent, childName, "", "")
		expectNamespaceParent(ctx, childName, sourceParent)

		targetSN := createSubNamespace(ctx, targetParent, childName, "", otherParent)
		Eventually(komega.Object(targetSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", haveCondition(
				accuratev2.SubNamespaceTypeMoveStalled,
				metav1.ConditionTrue,
				accuratev2.SubNamespaceReasonMoveTargetNotFound,
			)),
			HaveField("Status.Conditions", notHaveConditionType(string(kstatus.ConditionStalled))),
		))
		expectNamespaceParent(ctx, childName, sourceParent)

		Expect(komega.Update(sourceSN, func() {
			sourceSN.Spec.Move = &accuratev2.SubNamespaceMoveSpec{
				TargetParent: targetParent,
			}
		})()).To(Succeed())

		Eventually(komega.Object(sourceSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", haveCondition(
				accuratev2.SubNamespaceTypeMoveStalled,
				metav1.ConditionTrue,
				accuratev2.SubNamespaceReasonMoveNotAccepted,
			)),
			HaveField("Status.Conditions", notHaveConditionType(string(kstatus.ConditionStalled))),
		))
		Eventually(komega.Object(targetSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", haveCondition(
				accuratev2.SubNamespaceTypeMoveStalled,
				metav1.ConditionTrue,
				accuratev2.SubNamespaceReasonMoveTargetNotFound,
			)),
			HaveField("Status.Conditions", notHaveConditionType(string(kstatus.ConditionStalled))),
		))

		expectNamespaceParent(ctx, childName, sourceParent)
	})

	It("should move a sub-namespace after the target SubNamespace is created", func() {
		sourceParent := "test9-old"
		targetParent := "test9-new"
		childName := "test9-sub1"

		createNamespace(ctx, sourceParent)
		createNamespace(ctx, targetParent)

		sourceSN := createSubNamespace(ctx, sourceParent, childName, "", targetParent)
		expectNamespaceParent(ctx, childName, sourceParent)

		Eventually(komega.Object(sourceSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", notHaveConditionType(string(kstatus.ConditionStalled))),
			HaveField("Status.Conditions", haveCondition(
				accuratev2.SubNamespaceTypeMoveStalled,
				metav1.ConditionTrue,
				accuratev2.SubNamespaceReasonMoveTargetNotFound,
			)),
		))

		targetNS := createSubNamespace(ctx, targetParent, childName, sourceParent, "")
		expectNamespaceParent(ctx, childName, targetParent)

		Eventually(komega.Object(sourceSN)).Should(SatisfyAll(
			HaveField("Status.Conditions", haveCondition(
				string(kstatus.ConditionStalled),
				metav1.ConditionTrue,
				accuratev2.SubNamespaceReasonConflict,
			)),
			HaveField("Status.Conditions", notHaveConditionType(accuratev2.SubNamespaceTypeMoveStalled)),
		))

		Eventually(komega.Object(targetNS)).Should(SatisfyAll(
			HaveField("Status.Conditions", notHaveConditionType(string(kstatus.ConditionStalled))),
			HaveField("Status.Conditions", notHaveConditionType(accuratev2.SubNamespaceTypeMoveStalled)),
		))
	})
})

func createNamespace(ctx context.Context, name string) *corev1.Namespace {
	GinkgoHelper()

	ns := &corev1.Namespace{}
	ns.Name = name

	Expect(k8sClient.Create(ctx, ns)).To(Succeed())
	Eventually(komega.Get(ns)).Should(Succeed())

	return ns
}

func createSubNamespace(ctx context.Context, namespace, name, sourceParent, targetParent string) *accuratev2.SubNamespace {
	GinkgoHelper()

	sn := &accuratev2.SubNamespace{}
	sn.Namespace = namespace
	sn.Name = name

	if sourceParent != "" || targetParent != "" {
		sn.Spec.Move = &accuratev2.SubNamespaceMoveSpec{
			SourceParent: sourceParent,
			TargetParent: targetParent,
		}
	}

	Expect(k8sClient.Create(ctx, sn)).To(Succeed())
	Eventually(komega.Get(sn)).Should(Succeed())
	return sn
}

func expectNamespaceParent(ctx context.Context, name, expectedParent string) *corev1.Namespace {
	GinkgoHelper()

	ns := &corev1.Namespace{}
	ns.Name = name

	Eventually(komega.Get(ns)).Should(Succeed())
	Eventually(komega.Object(ns)).Should(
		HaveField("Labels", HaveKeyWithValue(constants.LabelParent, expectedParent)),
	)

	return ns
}

func haveCondition(conditionType string, status metav1.ConditionStatus, reason string) gomegaTypes.GomegaMatcher {
	return ContainElement(SatisfyAll(
		HaveField("Type", Equal(conditionType)),
		HaveField("Status", Equal(status)),
		HaveField("Reason", Equal(reason)),
	))
}

func notHaveConditionType(conditionType string) gomegaTypes.GomegaMatcher {
	return Not(ContainElement(
		HaveField("Type", Equal(conditionType)),
	))
}
