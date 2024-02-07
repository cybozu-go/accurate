package controllers

import (
	"context"
	"time"

	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/cybozu-go/accurate/pkg/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var _ = Describe("SubNamespace controller", func() {
	ctx := context.Background()
	var stopFunc func()

	const (
		rootNS     = "prop-root"
		sub1NS     = "prop-sub1"
		sub2NS     = "prop-sub2"
		sub1SubNS  = "prop-sub1-sub"
		tmplNS     = "prop-tmpl"
		instanceNS = "prop-instance"
	)

	svcRes := &unstructured.Unstructured{}
	svcRes.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   corev1.GroupName,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Service",
	})

	BeforeEach(func() {
		for _, ns := range []string{rootNS, sub1NS, sub2NS, sub1SubNS} {
			Expect(k8sClient.DeleteAllOf(ctx, &corev1.ConfigMap{}, client.InNamespace(ns))).To(Succeed())
		}
		svcList := &corev1.ServiceList{}
		Expect(k8sClient.List(ctx, svcList)).To(Succeed())
		for i := range svcList.Items {
			Expect(k8sClient.Delete(ctx, &svcList.Items[i])).To(Succeed())
		}

		mgr, err := ctrl.NewManager(k8sCfg, ctrl.Options{
			Scheme:         scheme,
			LeaderElection: false,
			Metrics:        server.Options{BindAddress: "0"},
		})
		Expect(err).ToNot(HaveOccurred())

		err = indexing.SetupIndexForNamespace(ctx, mgr)
		Expect(err).NotTo(HaveOccurred())
		err = indexing.SetupIndexForResource(ctx, mgr, svcRes)
		Expect(err).NotTo(HaveOccurred())

		pc := NewPropagateController(svcRes)
		err = pc.SetupWithManager(mgr)
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

	It("should propagate resources for mode=create", func() {
		By("creating a resource in the root namespace")
		svc1 := &corev1.Service{}
		svc1.Namespace = rootNS
		svc1.Name = "svc1"
		svc1.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateCreate}
		svc1.Spec.ClusterIP = "None"
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt32(3333)}}
		Expect(k8sClient.Create(ctx, svc1)).To(Succeed())
		Expect(komega.UpdateStatus(svc1, func() {
			svc1.Status.Conditions = []metav1.Condition{{
				Type:               "foo",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "Foo",
				Message:            "blah",
			}}
		})()).To(Succeed())

		svc2 := &corev1.Service{}
		svc2.Namespace = rootNS
		svc2.Name = "svc2"
		svc2.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt32(3333)}}
		Expect(k8sClient.Create(ctx, svc2)).To(Succeed())

		svc1Sub1 := &corev1.Service{}
		svc1Sub1.Name = "svc1"
		svc1Sub1.Namespace = sub1NS
		Eventually(komega.Get(svc1Sub1)).Should(Succeed())
		svc1Sub2 := &corev1.Service{}
		svc1Sub2.Name = "svc1"
		svc1Sub2.Namespace = sub2NS
		Eventually(komega.Get(svc1Sub2)).Should(Succeed())
		svc1Sub1Sub := &corev1.Service{}
		svc1Sub1Sub.Name = "svc1"
		svc1Sub1Sub.Namespace = sub1SubNS
		Eventually(komega.Get(svc1Sub1Sub)).Should(Succeed())
		Expect(svc1Sub1.Annotations).To(HaveKeyWithValue(constants.AnnFrom, rootNS))
		Expect(svc1Sub1.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub1.Spec.Ports[0].Port).To(BeNumerically("==", 3333))
		Expect(svc1Sub2.Annotations).To(HaveKeyWithValue(constants.AnnFrom, rootNS))
		Expect(svc1Sub2.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub2.Spec.Ports[0].Port).To(BeNumerically("==", 3333))
		Expect(svc1Sub1Sub.Annotations).To(HaveKeyWithValue(constants.AnnFrom, sub1NS))
		Expect(svc1Sub1Sub.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub1Sub.Spec.Ports[0].Port).To(BeNumerically("==", 3333))

		svc2Sub1 := &corev1.Service{}
		svc2Sub1.Name = "svc2"
		svc2Sub1.Namespace = sub1NS
		Expect(komega.Get(svc2Sub1)()).Should(WithTransform(apierrors.IsNotFound, BeTrue()))

		By("deleting a sub-resource to check that Accurate re-creates it")
		uid := svc1Sub2.UID
		Expect(k8sClient.Delete(ctx, svc1Sub2)).To(Succeed())
		Eventually(komega.Object(svc1Sub2)).Should(HaveField("UID", Not(Equal(uid))))

		By("updating a sub-resource to check that Accurate won't fix it")
		Expect(komega.Update(svc1Sub1, func() {
			svc1Sub1.Annotations["hoge"] = "fuga"
		})()).To(Succeed())

		Eventually(komega.Object(svc1Sub1)).Should(HaveField("Annotations", HaveKeyWithValue("hoge", "fuga")))
		Consistently(komega.Object(svc1Sub1)).Should(HaveField("Annotations", HaveKeyWithValue("hoge", "fuga")))

		By("changing a sub-resource to a non-sub-resource")
		Expect(komega.Update(svc1Sub1Sub, func() {
			svc1Sub1Sub.Annotations = map[string]string{
				constants.AnnPropagate: constants.PropagateUpdate,
			}
		})()).To(Succeed())

		Eventually(komega.Object(svc1Sub1Sub)).Should(HaveField("Annotations", HaveKeyWithValue(constants.AnnPropagate, constants.PropagateUpdate)))
		Consistently(komega.Object(svc1Sub1Sub)).Should(HaveField("Annotations", HaveKeyWithValue(constants.AnnPropagate, constants.PropagateUpdate)))
		Expect(svc1Sub1Sub.Annotations).NotTo(HaveKey(constants.AnnFrom))

		By("deleting the root resource")
		Expect(k8sClient.Delete(ctx, svc1)).To(Succeed())

		Consistently(komega.Get(svc1Sub1)).Should(Succeed())
		Expect(svc1Sub1.DeletionTimestamp).To(BeNil())
	})

	It("should propagate resources from a template namespace", func() {
		By("creating a resource in the template namespace")
		svc1 := &corev1.Service{}
		svc1.Namespace = tmplNS
		svc1.Name = "svc1"
		svc1.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateCreate}
		svc1.Spec.ClusterIP = "None"
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt32(3333)}}
		Expect(k8sClient.Create(ctx, svc1)).To(Succeed())
		Expect(komega.UpdateStatus(svc1, func() {
			svc1.Status.Conditions = []metav1.Condition{{
				Type:               "foo",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "Foo",
				Message:            "blah",
			}}
		})()).To(Succeed())

		svc2 := &corev1.Service{}
		svc2.Namespace = tmplNS
		svc2.Name = "svc2"
		svc2.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt32(3333)}}
		Expect(k8sClient.Create(ctx, svc2)).To(Succeed())

		svc1Instance := &corev1.Service{}
		svc1Instance.Name = "svc1"
		svc1Instance.Namespace = instanceNS
		Eventually(komega.Get(svc1Instance)).Should(Succeed())
		Expect(svc1Instance.Annotations).To(HaveKeyWithValue(constants.AnnFrom, tmplNS))
		Expect(svc1Instance.Spec.Ports).To(HaveLen(1))
		Expect(svc1Instance.Spec.Ports[0].Port).To(BeNumerically("==", 3333))

		svc2Instance := &corev1.Service{}
		svc2Instance.Name = "svc2"
		svc2Instance.Namespace = instanceNS
		Expect(komega.Get(svc2Instance)()).Should(WithTransform(apierrors.IsNotFound, BeTrue()))

		By("deleting a sub-resource to check that Accurate re-creates it")
		uid := svc1Instance.UID
		Expect(k8sClient.Delete(ctx, svc1Instance)).To(Succeed())
		Eventually(komega.Object(svc1Instance)).Should(HaveField("UID", Not(Equal(uid))))
	})

	It("should propagate resources for mode=update", func() {
		By("creating a resource in the root namespace")
		svc1 := &corev1.Service{}
		svc1.Namespace = rootNS
		svc1.Name = "svc1"
		svc1.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateUpdate}
		svc1.Spec.ClusterIP = "None"
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt32(3333)}}
		Expect(k8sClient.Create(ctx, svc1)).To(Succeed())
		Expect(komega.UpdateStatus(svc1, func() {
			svc1.Status.Conditions = []metav1.Condition{{
				Type:               "foo",
				Status:             metav1.ConditionTrue,
				LastTransitionTime: metav1.Now(),
				Reason:             "Foo",
				Message:            "blah",
			}}
		})()).To(Succeed())

		svc1Sub1 := &corev1.Service{}
		svc1Sub1.Name = "svc1"
		svc1Sub1.Namespace = sub1NS
		Eventually(komega.Get(svc1Sub1)).Should(Succeed())
		svc1Sub2 := &corev1.Service{}
		svc1Sub2.Name = "svc1"
		svc1Sub2.Namespace = sub2NS
		Eventually(komega.Get(svc1Sub2)).Should(Succeed())
		svc1Sub1Sub := &corev1.Service{}
		svc1Sub1Sub.Name = "svc1"
		svc1Sub1Sub.Namespace = sub1SubNS
		Eventually(komega.Get(svc1Sub1Sub)).Should(Succeed())
		Expect(svc1Sub1.Annotations).To(HaveKeyWithValue(constants.AnnFrom, rootNS))
		Expect(svc1Sub1.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub1.Spec.Ports[0].Port).To(BeNumerically("==", 3333))
		Expect(svc1Sub2.Annotations).To(HaveKeyWithValue(constants.AnnFrom, rootNS))
		Expect(svc1Sub2.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub2.Spec.Ports[0].Port).To(BeNumerically("==", 3333))
		Expect(svc1Sub1Sub.Annotations).To(HaveKeyWithValue(constants.AnnFrom, sub1NS))
		Expect(svc1Sub1Sub.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub1Sub.Spec.Ports[0].Port).To(BeNumerically("==", 3333))

		By("deleting a sub-resource to check that Accurate re-creates it")
		uid := svc1Sub2.UID
		Expect(k8sClient.Delete(ctx, svc1Sub2)).To(Succeed())
		Eventually(komega.Object(svc1Sub2)).Should(HaveField("UID", Not(Equal(uid))))

		By("updating a sub-resource to check that Accurate fixes it")
		Expect(komega.Update(svc1Sub1, func() {
			svc1Sub1.Annotations[constants.AnnPropagate] = constants.PropagateCreate
		})()).To(Succeed())
		rv := svc1Sub1.ResourceVersion
		rv2 := svc1Sub1Sub.ResourceVersion
		Eventually(komega.Object(svc1Sub1)).Should(HaveField("ResourceVersion", Not(Equal(rv))))
		Expect(svc1Sub1.Annotations).To(HaveKeyWithValue(constants.AnnPropagate, constants.PropagateUpdate))
		time.Sleep(100 * time.Millisecond)
		Expect(svc1Sub1Sub.ResourceVersion).To(Equal(rv2))

		By("updating a root resource to check that Accurate propagates the change to sub-resources")
		Expect(komega.Update(svc1, func() {
			svc1.Labels = map[string]string{"foo": "bar"}
		})()).To(Succeed())

		for _, ns := range []string{sub1NS, sub2NS, sub1SubNS} {
			svc := &corev1.Service{}
			svc.Name = "svc1"
			svc.Namespace = ns
			Eventually(komega.Object(svc)).Should(HaveField("Labels", HaveKeyWithValue("foo", "bar")))
		}

		By("deleting a root resource to check that Accurate cascades the deletion")
		Expect(k8sClient.Delete(ctx, svc1)).To(Succeed())

		for _, ns := range []string{sub1NS, sub2NS, sub1SubNS} {
			svc := &corev1.Service{}
			svc.Name = "svc1"
			svc.Namespace = ns
			Eventually(komega.Get(svc)).Should(WithTransform(apierrors.IsNotFound, BeTrue()))
		}
	})

	It("should manage generated resources", func() {
		cm1 := &corev1.ConfigMap{}
		cm1.Namespace = rootNS
		cm1.Name = "cm-generate"
		//lint:ignore SA1019 subject for removal
		cm1.Annotations = map[string]string{constants.AnnPropagateGenerated: constants.PropagateUpdate}
		cm1.Data = map[string]string{"foo": "bar"}
		Expect(k8sClient.Create(ctx, cm1)).To(Succeed())

		cm2 := &corev1.ConfigMap{}
		cm2.Namespace = rootNS
		cm2.Name = "cm-no-generate"
		cm2.Data = map[string]string{"abc": "def"}
		Expect(k8sClient.Create(ctx, cm2)).To(Succeed())

		svc1 := &corev1.Service{}
		svc1.Namespace = rootNS
		svc1.Name = "svc1"
		ctrl.SetControllerReference(cm1, svc1, scheme)
		svc1.Spec.ClusterIP = "None"
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt32(3333)}}
		Expect(k8sClient.Create(ctx, svc1)).To(Succeed())

		Eventually(komega.Object(svc1)).Should(HaveField("Annotations", HaveKeyWithValue(constants.AnnPropagate, constants.PropagateUpdate)))
		//lint:ignore SA1019 subject for removal
		Expect(svc1.Annotations).NotTo(HaveKey(constants.AnnGenerated))

		svc2 := &corev1.Service{}
		svc2.Namespace = rootNS
		svc2.Name = "svc2"
		Expect(ctrl.SetControllerReference(cm2, svc2, scheme)).To(Succeed())
		svc2.Spec.ClusterIP = "None"
		svc2.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt32(3333)}}
		Expect(k8sClient.Create(ctx, svc2)).To(Succeed())

		//lint:ignore SA1019 subject for removal
		Eventually(komega.Object(svc2)).Should(HaveField("Annotations", HaveKeyWithValue(constants.AnnGenerated, notGenerated)))
	})
})
