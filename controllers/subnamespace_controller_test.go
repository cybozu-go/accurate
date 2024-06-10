package controllers

import (
	"context"
	"time"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"github.com/cybozu-go/accurate/pkg/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
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
		})
		Expect(err).ToNot(HaveOccurred())

		snr := &SubNamespaceReconciler{
			Client: mgr.GetClient(),
		}
		err = snr.SetupWithManager(mgr)
		Expect(err).ToNot(HaveOccurred())

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

	It("should create and delete sub namespaces", func() {
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
		Expect(sn.Status.Conditions[0].Reason).To(Equal(accuratev2.SubNamespaceConflict))
	})

	It("should not delete a conflicting sub namespace", func() {
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
		Expect(sn.Status.Conditions[0].Reason).To(Equal(accuratev2.SubNamespaceConflict))

		Expect(k8sClient.Delete(ctx, sn)).To(Succeed())

		Consistently(komega.Object(sub1)).Should(HaveField("DeletionTimestamp", BeNil()))
	})

	It("should re-create a subnamespace if it is deleted", func() {
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
})
