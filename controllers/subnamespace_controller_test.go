package controllers

import (
	"context"
	"time"

	accuratev1 "github.com/cybozu-go/accurate/api/v1"
	"github.com/cybozu-go/accurate/pkg/constants"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("SubNamespace controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(k8sCfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
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
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sn := &accuratev1.SubNamespace{}
		sn.Namespace = "test1"
		sn.Name = "test1-sub1"
		sn.Finalizers = []string{constants.Finalizer}
		sn.Spec.Labels = map[string]string{
			"foo": "bar",
		}
		sn.Spec.Annotations = map[string]string{
			"foo": "bar",
		}
		err = k8sClient.Create(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		var sub1 *corev1.Namespace
		Eventually(func() error {
			sub1 = &corev1.Namespace{}
			return k8sClient.Get(ctx, client.ObjectKey{Name: "test1-sub1"}, sub1)
		}).Should(Succeed())

		Expect(sub1.Labels).To(HaveKeyWithValue(constants.LabelCreatedBy, "accurate"))
		Expect(sub1.Labels).To(HaveKeyWithValue(constants.LabelParent, "test1"))

		Expect(sub1.Labels).To(HaveKeyWithValue("foo", "bar"))
		Expect(sub1.Annotations).To(HaveKeyWithValue("foo", "bar"))

		Eventually(func() accuratev1.SubNamespaceStatus {
			sn = &accuratev1.SubNamespace{}
			err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "test1", Name: "test1-sub1"}, sn)
			if err != nil {
				return ""
			}
			return sn.Status
		}).Should(Equal(accuratev1.SubNamespaceOK))

		err = k8sClient.Delete(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() bool {
			sub1 = &corev1.Namespace{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "test1-sub1"}, sub1)
			if err != nil {
				return apierrors.IsNotFound(err)
			}
			return sub1.DeletionTimestamp != nil
		}).Should(BeTrue())
	})

	It("should detect conflicts", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test2"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		ns2 := &corev1.Namespace{}
		ns2.Name = "test2-sub1"
		err = k8sClient.Create(ctx, ns2)
		Expect(err).NotTo(HaveOccurred())

		sn := &accuratev1.SubNamespace{}
		sn.Namespace = "test2"
		sn.Name = "test2-sub1"
		err = k8sClient.Create(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() accuratev1.SubNamespaceStatus {
			sn = &accuratev1.SubNamespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "test2", Name: "test2-sub1"}, sn); err != nil {
				return ""
			}
			return sn.Status
		}).Should(Equal(accuratev1.SubNamespaceConflict))
	})

	It("should not delete a conflicting sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test3"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sn := &accuratev1.SubNamespace{}
		sn.Namespace = "test3"
		sn.Name = "test3-sub1"
		sn.Finalizers = []string{constants.Finalizer}
		err = k8sClient.Create(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		var sub1 *corev1.Namespace
		Eventually(func() error {
			sub1 = &corev1.Namespace{}
			return k8sClient.Get(ctx, client.ObjectKey{Name: "test3-sub1"}, sub1)
		}).Should(Succeed())

		sub1.Labels[constants.LabelParent] = "foo"
		err = k8sClient.Update(ctx, sub1)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() accuratev1.SubNamespaceStatus {
			sn = &accuratev1.SubNamespace{}
			err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "test3", Name: "test3-sub1"}, sn)
			if err != nil {
				return ""
			}
			return sn.Status
		}).Should(Equal(accuratev1.SubNamespaceConflict))

		err = k8sClient.Delete(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		Consistently(func() bool {
			sub1 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "test3-sub1"}, sub1); err != nil {
				return false
			}
			return sub1.DeletionTimestamp == nil
		}).Should(BeTrue())
	})

	It("should re-create a subnamespace if it is deleted", func() {
		ns := &corev1.Namespace{}
		ns.Name = "test4"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sn := &accuratev1.SubNamespace{}
		sn.Namespace = "test4"
		sn.Name = "test4-sub1"
		err = k8sClient.Create(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		var sub1 *corev1.Namespace
		Eventually(func() error {
			sub1 = &corev1.Namespace{}
			return k8sClient.Get(ctx, client.ObjectKey{Name: "test4-sub1"}, sub1)
		}).Should(Succeed())

		err = k8sClient.Delete(ctx, sub1)
		Expect(err).NotTo(HaveOccurred())

		uid := sub1.UID
		sub1 = &corev1.Namespace{}
		sub1.Name = "test4-sub1"
		cs, err := kubernetes.NewForConfig(k8sCfg)
		Expect(err).NotTo(HaveOccurred())
		_, err = cs.CoreV1().Namespaces().Finalize(ctx, sub1, metav1.UpdateOptions{})
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() bool {
			sub1 = &corev1.Namespace{}
			err := k8sClient.Get(ctx, client.ObjectKey{Name: "test4-sub1"}, sub1)
			if err != nil {
				return false
			}
			return sub1.UID != uid
		}).Should(BeTrue())
	})
})
