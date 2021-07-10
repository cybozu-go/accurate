package controllers

import (
	"context"
	"time"

	"github.com/cybozu-go/innu/pkg/constants"
	"github.com/cybozu-go/innu/pkg/indexing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			err := k8sClient.DeleteAllOf(ctx, &corev1.ConfigMap{}, client.InNamespace(ns))
			Expect(err).NotTo(HaveOccurred())
		}
		svcList := &corev1.ServiceList{}
		err := k8sClient.List(ctx, svcList)
		Expect(err).NotTo(HaveOccurred())
		for i := range svcList.Items {
			err := k8sClient.Delete(ctx, &svcList.Items[i])
			Expect(err).NotTo(HaveOccurred())
		}

		mgr, err := ctrl.NewManager(k8sCfg, ctrl.Options{
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
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
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt(3333)}}
		err := k8sClient.Create(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())
		svc1.Status.Conditions = []metav1.Condition{{
			Type:               "foo",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "Foo",
			Message:            "blah",
		}}
		err = k8sClient.Status().Update(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())

		svc2 := &corev1.Service{}
		svc2.Namespace = rootNS
		svc2.Name = "svc2"
		svc2.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt(3333)}}
		err = k8sClient.Create(ctx, svc2)
		Expect(err).NotTo(HaveOccurred())

		var svc1Sub1, svc1Sub2, svc1Sub1Sub *corev1.Service
		Eventually(func() error {
			svc1Sub1 = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1NS, Name: "svc1"}, svc1Sub1)
		}).Should(Succeed())
		Eventually(func() error {
			svc1Sub2 = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: sub2NS, Name: "svc1"}, svc1Sub2)
		}).Should(Succeed())
		Eventually(func() error {
			svc1Sub1Sub = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1SubNS, Name: "svc1"}, svc1Sub1Sub)
		}).Should(Succeed())
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
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1NS, Name: "svc2"}, svc2Sub1)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())

		By("deleting a sub-resource to check that Innu re-creates it")
		uid := svc1Sub2.UID
		err = k8sClient.Delete(ctx, svc1Sub2)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() types.UID {
			svc := &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: sub2NS, Name: "svc1"}, svc); err != nil {
				return uid
			}
			return svc.UID
		}).ShouldNot(Equal(uid))

		By("updating a sub-resource to check that Innu won't fix it")
		svc1Sub1.Annotations["hoge"] = "fuga"
		err = k8sClient.Update(ctx, svc1Sub1)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			svc1Sub1 = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1NS, Name: "svc1"}, svc1Sub1); err != nil {
				return ""
			}
			return svc1Sub1.Annotations["hoge"]
		}).Should(Equal("fuga"))
		Consistently(func() string {
			svc1Sub1 = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1NS, Name: "svc1"}, svc1Sub1); err != nil {
				return ""
			}
			return svc1Sub1.Annotations["hoge"]
		}).Should(Equal("fuga"))

		By("changing a sub-resource to a non-sub-resource")
		svc1Sub1Sub.Annotations = map[string]string{
			constants.AnnPropagate: constants.PropagateUpdate,
		}
		err = k8sClient.Update(ctx, svc1Sub1Sub)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			svc1Sub1Sub = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1SubNS, Name: "svc1"}, svc1Sub1Sub); err != nil {
				return ""
			}
			return svc1Sub1Sub.Annotations[constants.AnnPropagate]
		}).Should(Equal(constants.PropagateUpdate))
		Consistently(func() string {
			svc1Sub1Sub = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1SubNS, Name: "svc1"}, svc1Sub1Sub); err != nil {
				return ""
			}
			return svc1Sub1Sub.Annotations[constants.AnnPropagate]
		}).Should(Equal(constants.PropagateUpdate))
		Expect(svc1Sub1Sub.Annotations).NotTo(HaveKey(constants.AnnFrom))

		By("deleting the root resource")
		err = k8sClient.Delete(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())

		Consistently(func() error {
			svc1Sub1 = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1NS, Name: "svc1"}, svc1Sub1)
		}).Should(Succeed())
		Expect(svc1Sub1.DeletionTimestamp).To(BeNil())
	})

	It("should propagate resources from a template namespace", func() {
		By("creating a resource in the template namespace")
		svc1 := &corev1.Service{}
		svc1.Namespace = tmplNS
		svc1.Name = "svc1"
		svc1.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateCreate}
		svc1.Spec.ClusterIP = "None"
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt(3333)}}
		err := k8sClient.Create(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())
		svc1.Status.Conditions = []metav1.Condition{{
			Type:               "foo",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "Foo",
			Message:            "blah",
		}}
		err = k8sClient.Status().Update(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())

		svc2 := &corev1.Service{}
		svc2.Namespace = tmplNS
		svc2.Name = "svc2"
		svc2.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt(3333)}}
		err = k8sClient.Create(ctx, svc2)
		Expect(err).NotTo(HaveOccurred())

		var svc1Instance *corev1.Service
		Eventually(func() error {
			svc1Instance = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: instanceNS, Name: "svc1"}, svc1Instance)
		}).Should(Succeed())
		Expect(svc1Instance.Annotations).To(HaveKeyWithValue(constants.AnnFrom, tmplNS))
		Expect(svc1Instance.Spec.Ports).To(HaveLen(1))
		Expect(svc1Instance.Spec.Ports[0].Port).To(BeNumerically("==", 3333))

		svc2Instance := &corev1.Service{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: instanceNS, Name: "svc2"}, svc2Instance)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())

		By("deleting a sub-resource to check that Innu re-creates it")
		uid := svc1Instance.UID
		err = k8sClient.Delete(ctx, svc1Instance)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() types.UID {
			svc1Instance = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: instanceNS, Name: "svc1"}, svc1Instance); err != nil {
				return uid
			}
			return svc1Instance.UID
		}).ShouldNot(Equal(uid))
	})

	It("should propagate resources for mode=update", func() {
		By("creating a resource in the root namespace")
		svc1 := &corev1.Service{}
		svc1.Namespace = rootNS
		svc1.Name = "svc1"
		svc1.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateUpdate}
		svc1.Spec.ClusterIP = "None"
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt(3333)}}
		err := k8sClient.Create(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())
		svc1.Status.Conditions = []metav1.Condition{{
			Type:               "foo",
			Status:             metav1.ConditionTrue,
			LastTransitionTime: metav1.Now(),
			Reason:             "Foo",
			Message:            "blah",
		}}
		err = k8sClient.Status().Update(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())

		var svc1Sub1, svc1Sub2, svc1Sub1Sub *corev1.Service
		Eventually(func() error {
			svc1Sub1 = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1NS, Name: "svc1"}, svc1Sub1)
		}).Should(Succeed())
		Eventually(func() error {
			svc1Sub2 = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: sub2NS, Name: "svc1"}, svc1Sub2)
		}).Should(Succeed())
		Eventually(func() error {
			svc1Sub1Sub = &corev1.Service{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1SubNS, Name: "svc1"}, svc1Sub1Sub)
		}).Should(Succeed())
		Expect(svc1Sub1.Annotations).To(HaveKeyWithValue(constants.AnnFrom, rootNS))
		Expect(svc1Sub1.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub1.Spec.Ports[0].Port).To(BeNumerically("==", 3333))
		Expect(svc1Sub2.Annotations).To(HaveKeyWithValue(constants.AnnFrom, rootNS))
		Expect(svc1Sub2.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub2.Spec.Ports[0].Port).To(BeNumerically("==", 3333))
		Expect(svc1Sub1Sub.Annotations).To(HaveKeyWithValue(constants.AnnFrom, sub1NS))
		Expect(svc1Sub1Sub.Spec.Ports).To(HaveLen(1))
		Expect(svc1Sub1Sub.Spec.Ports[0].Port).To(BeNumerically("==", 3333))

		By("deleting a sub-resource to check that Innu re-creates it")
		uid := svc1Sub2.UID
		err = k8sClient.Delete(ctx, svc1Sub2)
		Expect(err).NotTo(HaveOccurred())
		Eventually(func() types.UID {
			svc := &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: sub2NS, Name: "svc1"}, svc); err != nil {
				return uid
			}
			return svc.UID
		}).ShouldNot(Equal(uid))

		By("updating a sub-resource to check that Innu fixes it")
		svc1Sub1.Annotations[constants.AnnPropagate] = constants.PropagateCreate
		err = k8sClient.Update(ctx, svc1Sub1)
		Expect(err).NotTo(HaveOccurred())
		rv := svc1Sub1.ResourceVersion
		rv2 := svc1Sub1Sub.ResourceVersion
		Eventually(func() string {
			svc1Sub1 = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: sub1NS, Name: "svc1"}, svc1Sub1); err != nil {
				return rv
			}
			return svc1Sub1.ResourceVersion
		}).ShouldNot(Equal(rv))
		Expect(svc1Sub1.Annotations).To(HaveKeyWithValue(constants.AnnPropagate, constants.PropagateUpdate))
		time.Sleep(100 * time.Millisecond)
		Expect(svc1Sub1Sub.ResourceVersion).To(Equal(rv2))

		By("updating a root resource to check that Innu propagates the change to sub-resources")
		svc1.Labels = map[string]string{"foo": "bar"}
		err = k8sClient.Update(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())

		for _, ns := range []string{sub1NS, sub2NS, sub1SubNS} {
			Eventually(func() string {
				svc := &corev1.Service{}
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: "svc1"}, svc); err != nil {
					return ""
				}
				return svc.Labels["foo"]
			}).Should(Equal("bar"))
		}

		By("deleting a root resource to check that Innu cascades the deletion")
		err = k8sClient.Delete(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())

		for _, ns := range []string{sub1NS, sub2NS, sub1SubNS} {
			Eventually(func() bool {
				svc := &corev1.Service{}
				if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: "svc1"}, svc); err != nil {
					return apierrors.IsNotFound(err)
				}
				return false
			}).Should(BeTrue())
		}
	})

	It("should manage generated resources", func() {
		cm1 := &corev1.ConfigMap{}
		cm1.Namespace = rootNS
		cm1.Name = "cm-generate"
		cm1.Annotations = map[string]string{constants.AnnPropagateGenerated: constants.PropagateUpdate}
		cm1.Data = map[string]string{"foo": "bar"}
		err := k8sClient.Create(ctx, cm1)
		Expect(err).NotTo(HaveOccurred())

		cm2 := &corev1.ConfigMap{}
		cm2.Namespace = rootNS
		cm2.Name = "cm-no-generate"
		cm2.Data = map[string]string{"abc": "def"}
		err = k8sClient.Create(ctx, cm2)
		Expect(err).NotTo(HaveOccurred())

		svc1 := &corev1.Service{}
		svc1.Namespace = rootNS
		svc1.Name = "svc1"
		ctrl.SetControllerReference(cm1, svc1, scheme)
		svc1.Spec.ClusterIP = "None"
		svc1.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt(3333)}}
		err = k8sClient.Create(ctx, svc1)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			svc1 = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: rootNS, Name: "svc1"}, svc1); err != nil {
				return ""
			}
			return svc1.Annotations[constants.AnnPropagate]
		}).Should(Equal(constants.PropagateUpdate))
		Expect(svc1.Annotations).NotTo(HaveKey(constants.AnnGenerated))

		svc2 := &corev1.Service{}
		svc2.Namespace = rootNS
		svc2.Name = "svc2"
		ctrl.SetControllerReference(cm2, svc2, scheme)
		svc2.Spec.ClusterIP = "None"
		svc2.Spec.Ports = []corev1.ServicePort{{Port: 3333, TargetPort: intstr.FromInt(3333)}}
		err = k8sClient.Create(ctx, svc2)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			svc2 = &corev1.Service{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Namespace: rootNS, Name: "svc2"}, svc2); err != nil {
				return ""
			}
			return svc2.Annotations[constants.AnnGenerated]
		}).Should(Equal(notGenerated))
	})
})
