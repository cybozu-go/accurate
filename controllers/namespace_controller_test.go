package controllers

import (
	"context"
	"time"

	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/constants"
	"github.com/cybozu-go/accurate/pkg/indexing"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest/komega"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var roleRes, secretRes *unstructured.Unstructured

func init() {
	roleRes = &unstructured.Unstructured{}
	roleRes.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   rbacv1.GroupName,
		Version: rbacv1.SchemeGroupVersion.Version,
		Kind:    "Role",
	})

	secretRes = &unstructured.Unstructured{}
	secretRes.SetGroupVersionKind(schema.GroupVersionKind{
		Group:   corev1.GroupName,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "Secret",
	})
}

var _ = Describe("Namespace controller", func() {
	ctx := context.Background()
	var stopFunc func()

	BeforeEach(func() {
		mgr, err := ctrl.NewManager(k8sCfg, ctrl.Options{
			Scheme:         scheme,
			LeaderElection: false,
			Metrics:        server.Options{BindAddress: "0"},
			Client: client.Options{
				Cache: &client.CacheOptions{
					Unstructured: true,
				},
			},
		})
		Expect(err).ToNot(HaveOccurred())

		nr := &NamespaceReconciler{
			Client:                     mgr.GetClient(),
			LabelKeys:                  []string{"foo.bar/baz", "team", "*.glob/*"},
			AnnotationKeys:             []string{"foo.bar/zot", "memo", "*.glob/*"},
			SubNamespaceLabelKeys:      []string{"foo.bar/baz", "team", "*.glob/*"},
			SubNamespaceAnnotationKeys: []string{"foo.bar/zot", "memo", "*.glob/*"},
			Watched:                    []*unstructured.Unstructured{roleRes, secretRes},
		}
		err = nr.SetupWithManager(mgr)
		Expect(err).ToNot(HaveOccurred())

		err = indexing.SetupIndexForNamespace(ctx, mgr)
		Expect(err).NotTo(HaveOccurred())
		err = indexing.SetupIndexForResource(ctx, mgr, roleRes)
		Expect(err).NotTo(HaveOccurred())
		err = indexing.SetupIndexForResource(ctx, mgr, secretRes)
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

	It("should implement template namespace correctly", func() {
		tmpl := &corev1.Namespace{}
		tmpl.Name = "tmpl"
		tmpl.Labels = map[string]string{
			constants.LabelType: constants.NSTypeTemplate,
			"foo.bar/baz":       "baz",
			"team":              "neco",
			"memo":              "randum",
		}
		tmpl.Annotations = map[string]string{
			"foo.bar/zot": "zot",
			"memo":        "memo",
			"team":        "cat",
		}
		Expect(k8sClient.Create(ctx, tmpl)).To(Succeed())

		role1 := roleRes.DeepCopy()
		role1.SetNamespace("tmpl")
		role1.SetName("role1")
		role1.Object["rules"] = []interface{}{
			map[string]interface{}{
				"apiGroups": []interface{}{""},
				"resources": []interface{}{"pods"},
				"verbs":     []interface{}{"get", "watch", "list"},
			},
		}
		Expect(k8sClient.Create(ctx, role1)).To(Succeed())

		role2 := roleRes.DeepCopy()
		role2.SetNamespace("tmpl")
		role2.SetName("role2")
		role2.SetAnnotations(map[string]string{constants.AnnPropagate: constants.PropagateCreate})
		role2.Object["rules"] = []interface{}{
			map[string]interface{}{
				"apiGroups": []interface{}{""},
				"resources": []interface{}{"pods"},
				"verbs":     []interface{}{"get", "watch", "list"},
			},
		}
		Expect(k8sClient.Create(ctx, role2)).To(Succeed())

		secret := secretRes.DeepCopy()
		secret.SetNamespace("tmpl")
		secret.SetName("foo")
		secret.Object["data"] = map[string]interface{}{
			"foo": "MjAyMC0wOS0xM1QwNDozOToxMFo=",
		}
		secret.SetAnnotations(map[string]string{constants.AnnPropagate: constants.PropagateUpdate})
		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		ns1 := &corev1.Namespace{}
		ns1.Name = "ns1"
		Expect(k8sClient.Create(ctx, ns1)).To(Succeed())

		secret2 := secretRes.DeepCopy()
		secret2.SetNamespace("ns1")
		secret2.SetName("bar")
		secret2.Object["data"] = map[string]interface{}{
			"bar": "MjAyMC0wOS0xM1QwNDozOToxMFo=",
		}
		secret2.SetAnnotations(map[string]string{constants.AnnPropagate: constants.PropagateUpdate})
		Expect(k8sClient.Create(ctx, secret2)).To(Succeed())

		time.Sleep(100 * time.Millisecond)

		By("setting the template namespace")
		Expect(komega.Update(ns1, func() {
			ns1.Labels = map[string]string{constants.LabelTemplate: "tmpl"}
		})()).To(Succeed())

		Eventually(komega.Object(ns1)).Should(HaveField("Labels", HaveKeyWithValue("team", "neco")))
		Expect(ns1.Labels).To(HaveKeyWithValue("foo.bar/baz", "baz"))
		Expect(ns1.Labels).NotTo(HaveKey("memo"))
		Expect(ns1.Annotations).To(HaveKeyWithValue("foo.bar/zot", "zot"))
		Expect(ns1.Annotations).To(HaveKeyWithValue("memo", "memo"))
		Expect(ns1.Annotations).NotTo(HaveKey("team"))

		pRole := &rbacv1.Role{}
		pRole.Name = "role2"
		pRole.Namespace = ns1.Name
		Eventually(komega.Get(pRole)).Should(Succeed())
		Expect(pRole.Labels).To(HaveKeyWithValue(constants.LabelCreatedBy, constants.CreatedBy))
		Expect(pRole.Annotations).To(HaveKeyWithValue(constants.AnnFrom, "tmpl"))
		Expect(pRole.Annotations).To(HaveKeyWithValue(constants.AnnPropagate, constants.PropagateCreate))
		Expect(pRole.Rules).To(HaveLen(1))

		pSecret := &corev1.Secret{}
		pSecret.Name = "foo"
		pSecret.Namespace = ns1.Name
		Eventually(komega.Get(pSecret)).Should(Succeed())
		Expect(pSecret.Labels).To(HaveKeyWithValue(constants.LabelCreatedBy, constants.CreatedBy))
		Expect(pSecret.Annotations).To(HaveKeyWithValue(constants.AnnFrom, "tmpl"))
		Expect(pSecret.Annotations).To(HaveKeyWithValue(constants.AnnPropagate, constants.PropagateUpdate))
		Expect(pSecret.Data).To(HaveKey("foo"))

		pRole2 := &rbacv1.Role{}
		pRole2.Name = "role1"
		pRole2.Namespace = ns1.Name
		Expect(komega.Get(pRole2)()).To(WithTransform(apierrors.IsNotFound, BeTrue()))

		pSecret2 := &corev1.Secret{}
		pSecret2.Name = "bar"
		pSecret2.Namespace = ns1.Name
		Expect(komega.Get(pSecret2)()).To(Succeed())

		By("changing a label of template namespace")
		Expect(komega.Update(tmpl, func() {
			tmpl.Labels["foo.bar/baz"] = "123"
		})()).To(Succeed())

		Eventually(komega.Object(ns1)).Should(HaveField("Labels", HaveKeyWithValue("foo.bar/baz", "123")))

		tmpl2 := &corev1.Namespace{}
		tmpl2.Name = "tmpl2"
		tmpl2.Labels = map[string]string{
			constants.LabelType: constants.NSTypeTemplate,
			"team":              "maneki",
		}
		Expect(k8sClient.Create(ctx, tmpl2)).To(Succeed())

		sec2 := &corev1.Secret{}
		sec2.Namespace = tmpl2.Name
		sec2.Name = "sec2"
		sec2.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateUpdate}
		sec2.Data = map[string][]byte{"foo": []byte("barbar")}
		Expect(k8sClient.Create(ctx, sec2)).To(Succeed())

		By("changing the template namespace to tmpl2")
		Expect(komega.Update(ns1, func() {
			ns1.Labels[constants.LabelTemplate] = "tmpl2"
		})()).To(Succeed())

		Eventually(komega.Get(pSecret)).Should(WithTransform(apierrors.IsNotFound, BeTrue()))

		pSec2 := &corev1.Secret{}
		pSec2.Name = "sec2"
		pSec2.Namespace = ns1.Name
		Eventually(komega.Get(pSec2)).Should(Succeed())

		Eventually(komega.Object(ns1)).Should(HaveField("Labels", HaveKeyWithValue("team", "maneki")))

		By("unsetting the template")
		Expect(komega.Update(ns1, func() {
			delete(ns1.Labels, constants.LabelTemplate)
		})()).To(Succeed())

		Eventually(komega.Get(pSec2)).Should(WithTransform(apierrors.IsNotFound, BeTrue()))
	})

	It("should handle propagation between template namespaces", func() {
		tmpl1 := &corev1.Namespace{}
		tmpl1.Name = "tree-tmpl-1"
		tmpl1.Labels = map[string]string{
			constants.LabelType: constants.NSTypeTemplate,
			"team":              "neco",
		}
		Expect(k8sClient.Create(ctx, tmpl1)).To(Succeed())

		tmpl2 := &corev1.Namespace{}
		tmpl2.Name = "tree-tmpl-2"
		tmpl2.Labels = map[string]string{
			constants.LabelType:     constants.NSTypeTemplate,
			constants.LabelTemplate: "tree-tmpl-1",
		}
		tmpl2.Annotations = map[string]string{"memo": "mome"}
		Expect(k8sClient.Create(ctx, tmpl2)).To(Succeed())

		instance := &corev1.Namespace{}
		instance.Name = "tree-instance"
		instance.Labels = map[string]string{
			constants.LabelTemplate: "tree-tmpl-2",
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		Eventually(komega.Object(instance)).Should(And(
			HaveField("Labels", HaveKeyWithValue("team", "neco")),
			HaveField("Annotations", HaveKeyWithValue("memo", "mome")),
		))

		Expect(komega.Update(tmpl2, func() {
			tmpl2.Labels["team"] = "hoge"
			tmpl2.Annotations["memo"] = "test"
		})()).To(Succeed())

		Consistently(komega.Object(instance)).Should(HaveField("Labels", HaveKeyWithValue("team", "neco")))

		Expect(instance.Annotations["memo"]).Should(Equal("test"))

		Expect(komega.Object(tmpl2)()).To(HaveField("Labels", HaveKeyWithValue("team", "neco")))
	})

	It("should not delete resources in an independent namespace", func() {
		secret := &corev1.Secret{}
		secret.Namespace = "default"
		secret.Name = "independent"
		secret.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateUpdate}
		secret.Data = map[string][]byte{"foo": []byte("bar")}

		Expect(k8sClient.Create(ctx, secret)).To(Succeed())

		time.Sleep(100 * time.Millisecond)

		ns := &corev1.Namespace{}
		ns.Name = "default"
		Expect(komega.Update(ns, func() {
			if ns.Labels == nil {
				ns.Labels = make(map[string]string)
			}
			ns.Labels["accurate-test"] = "test"
		})()).To(Succeed())

		Consistently(komega.Get(secret), 1, 0.1).Should(Succeed())
	})

	It("should implement a sub namespace correctly", func() {
		root := &corev1.Namespace{}
		root.Name = "root"
		root.Labels = map[string]string{
			constants.LabelType:        constants.NSTypeRoot,
			"team":                     "neco",
			"foo.glob/a":               "glob",
			"do.not.match/glob.patten": "glob",
		}
		root.Annotations = map[string]string{
			"foo":                      "bar",
			"bar.glob/b":               "glob",
			"baz.glob/c":               "delete-me",
			"do.not.match/glob.patten": "glob",
		}
		Expect(k8sClient.Create(ctx, root)).To(Succeed())

		sec1 := &corev1.Secret{}
		sec1.Namespace = "root"
		sec1.Name = "sec1"
		sec1.Data = map[string][]byte{"foo": []byte("bar")}
		Expect(k8sClient.Create(ctx, sec1)).To(Succeed())

		sec2 := &corev1.Secret{}
		sec2.Namespace = "root"
		sec2.Name = "sec2"
		sec2.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateCreate}
		sec2.Data = map[string][]byte{"foo": []byte("bar")}
		Expect(k8sClient.Create(ctx, sec2)).To(Succeed())

		sec3 := &corev1.Secret{}
		sec3.Namespace = "root"
		sec3.Name = "sec3"
		sec3.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateUpdate}
		sec3.Data = map[string][]byte{"foo": []byte("bar")}
		Expect(k8sClient.Create(ctx, sec3)).To(Succeed())

		By("creating a sub namespace")
		sub1 := &corev1.Namespace{}
		sub1.Name = "sub1"
		sub1.Labels = map[string]string{constants.LabelParent: "root"}
		Expect(k8sClient.Create(ctx, sub1)).To(Succeed())

		Eventually(komega.Object(sub1)).Should(HaveField("Labels", HaveKeyWithValue("team", "neco")))
		Expect(sub1.Labels).Should(HaveKeyWithValue("foo.glob/a", "glob"))
		Expect(sub1.Labels).NotTo(HaveKey(constants.LabelType))
		Expect(sub1.Labels).NotTo(HaveKey("do.not.match/glob/patten"))
		Expect(sub1.Annotations).Should(HaveKeyWithValue("bar.glob/b", "glob"))
		Expect(sub1.Annotations).Should(HaveKeyWithValue("baz.glob/c", "delete-me"))
		Expect(sub1.Annotations).NotTo(HaveKey("foo"))
		Expect(sub1.Annotations).NotTo(HaveKey("do.not.match/glob/patten"))

		cSec2 := &corev1.Secret{}
		cSec2.Name = "sec2"
		cSec2.Namespace = sub1.Name
		Eventually(komega.Get(cSec2)).Should(Succeed())
		cSec3 := &corev1.Secret{}
		cSec3.Name = "sec3"
		cSec3.Namespace = sub1.Name
		Eventually(komega.Get(cSec3)).Should(Succeed())

		By("creating a grandchild namespace")
		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2"
		sub2.Labels = map[string]string{constants.LabelParent: "sub1"}
		Expect(k8sClient.Create(ctx, sub2)).To(Succeed())

		Eventually(komega.Object(sub2)).Should(HaveField("Labels", HaveKeyWithValue("team", "neco")))
		gcSec2 := &corev1.Secret{}
		gcSec2.Name = sec2.Name
		gcSec2.Namespace = sub2.Name
		Eventually(komega.Get(gcSec2)).Should(Succeed())
		gcSec3 := &corev1.Secret{}
		gcSec3.Name = sec3.Name
		gcSec3.Namespace = sub2.Name
		Eventually(komega.Get(gcSec3)).Should(Succeed())

		By("editing a label of root namespace")
		Expect(komega.Update(root, func() {
			root.Labels["team"] = "nuco"
		})()).To(Succeed())

		Eventually(komega.Object(sub1)).Should(HaveField("Labels", HaveKeyWithValue("team", "nuco")))
		Eventually(komega.Object(sub2)).Should(HaveField("Labels", HaveKeyWithValue("team", "nuco")))

		By("deleting an annotation in root namespace")
		Expect(komega.Update(root, func() {
			delete(root.Labels, "baz.glob/c")
		})()).To(Succeed())
		// Cleaning up obsolete labels/annotations from sub-namespaces is currently unsupported
		// See https://github.com/cybozu-go/accurate/issues/98
		Consistently(komega.Object(sub1)).Should(HaveField("Annotations", HaveKey("baz.glob/c")))
		//Eventually(komega.Object(sub1)).Should(HaveField("Annotations", Not(HaveKey("baz.glob/c"))))

		By("changing the parent of sub2")
		root2 := &corev1.Namespace{}
		root2.Name = "root2"
		root2.Labels = map[string]string{
			constants.LabelType: constants.NSTypeRoot,
			"foo.bar/baz":       "baz",
		}
		Expect(k8sClient.Create(ctx, root2)).To(Succeed())

		Expect(komega.Update(sub2, func() {
			sub2.Labels[constants.LabelParent] = "root2"
		})()).To(Succeed())

		Eventually(komega.Object(sub2)).Should(HaveField("Labels", HaveKeyWithValue("foo.bar/baz", "baz")))

		Eventually(komega.Get(gcSec3)).Should(WithTransform(apierrors.IsNotFound, BeTrue()))

		Expect(komega.Get(gcSec2)()).To(Succeed())

		By("creating a SubNamespace for sub1 namespace")
		sn := &accuratev2alpha1.SubNamespace{}
		sn.Namespace = "root"
		sn.Name = "sub1"
		sn.Spec.Labels = map[string]string{
			"team":  "neco",
			"empty": "true",
		}
		sn.Spec.Annotations = map[string]string{
			"memo":  "neco",
			"empty": "true",
		}
		sn.Finalizers = []string{constants.Finalizer}
		Expect(k8sClient.Create(ctx, sn)).To(Succeed())
		Eventually(komega.Object(sub1)).Should(And(
			HaveField("Labels", HaveKeyWithValue("team", "neco")),
			HaveField("Labels", Not(HaveKey("empty"))),
			HaveField("Annotations", HaveKeyWithValue("memo", "neco")),
			HaveField("Annotations", Not(HaveKey("empty"))),
		))

		By("returning the parent of sub2")
		Expect(komega.Update(sub2, func() {
			sub2.Labels[constants.LabelParent] = "sub1"
		})()).To(Succeed())
		Eventually(komega.Object(sub2)).Should(And(
			HaveField("Labels", HaveKeyWithValue("team", "neco")),
			HaveField("Labels", Not(HaveKey("empty"))),
			HaveField("Annotations", HaveKeyWithValue("memo", "neco")),
			HaveField("Annotations", Not(HaveKey("empty"))),
		))

		By("updating labels and annotations of SubNamespace for sub1 namespace")
		Expect(komega.Update(sn, func() {
			sn.Spec.Labels["team"] = "tama"
			sn.Spec.Annotations["memo"] = "tama"
		})()).To(Succeed())
		Eventually(komega.Object(sub1)).Should(And(
			HaveField("Labels", HaveKeyWithValue("team", "tama")),
			HaveField("Annotations", HaveKeyWithValue("memo", "tama")),
		))
		Eventually(komega.Object(sub2)).Should(And(
			HaveField("Labels", HaveKeyWithValue("team", "tama")),
			HaveField("Annotations", HaveKeyWithValue("memo", "tama")),
		))
	})
})
