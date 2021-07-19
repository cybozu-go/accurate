package controllers

import (
	"context"
	"time"

	"github.com/cybozu-go/innu/pkg/cluster"
	"github.com/cybozu-go/innu/pkg/constants"
	"github.com/cybozu-go/innu/pkg/indexing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			Scheme:             scheme,
			LeaderElection:     false,
			MetricsBindAddress: "0",
			NewClient:          cluster.NewCachingClient,
		})
		Expect(err).ToNot(HaveOccurred())

		nr := &NamespaceReconciler{
			Client:         mgr.GetClient(),
			LabelKeys:      []string{"foo.bar/baz", "team"},
			AnnotationKeys: []string{"foo.bar/zot", "memo"},
			Watched:        []*unstructured.Unstructured{roleRes, secretRes},
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
		err := k8sClient.Create(ctx, tmpl)
		Expect(err).NotTo(HaveOccurred())

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
		err = k8sClient.Create(ctx, role1)
		Expect(err).NotTo(HaveOccurred())

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
		err = k8sClient.Create(ctx, role2)
		Expect(err).NotTo(HaveOccurred())

		secret := secretRes.DeepCopy()
		secret.SetNamespace("tmpl")
		secret.SetName("foo")
		secret.Object["data"] = map[string]interface{}{
			"foo": "MjAyMC0wOS0xM1QwNDozOToxMFo=",
		}
		secret.SetAnnotations(map[string]string{constants.AnnPropagate: constants.PropagateUpdate})
		err = k8sClient.Create(ctx, secret)
		Expect(err).NotTo(HaveOccurred())

		ns1 := &corev1.Namespace{}
		ns1.Name = "ns1"
		err = k8sClient.Create(ctx, ns1)
		Expect(err).NotTo(HaveOccurred())

		secret2 := secretRes.DeepCopy()
		secret2.SetNamespace("ns1")
		secret2.SetName("bar")
		secret2.Object["data"] = map[string]interface{}{
			"bar": "MjAyMC0wOS0xM1QwNDozOToxMFo=",
		}
		secret2.SetAnnotations(map[string]string{constants.AnnPropagate: constants.PropagateUpdate})
		err = k8sClient.Create(ctx, secret2)
		Expect(err).NotTo(HaveOccurred())

		time.Sleep(100 * time.Millisecond)

		By("setting the template namespace")
		ns1.Labels = map[string]string{constants.LabelTemplate: "tmpl"}
		err = k8sClient.Update(ctx, ns1)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			ns1 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "ns1"}, ns1); err != nil {
				return ""
			}
			return ns1.Labels["team"]
		}).Should(Equal("neco"))
		Expect(ns1.Labels).To(HaveKeyWithValue("foo.bar/baz", "baz"))
		Expect(ns1.Labels).NotTo(HaveKey("memo"))
		Expect(ns1.Annotations).To(HaveKeyWithValue("foo.bar/zot", "zot"))
		Expect(ns1.Annotations).To(HaveKeyWithValue("memo", "memo"))
		Expect(ns1.Annotations).NotTo(HaveKey("team"))

		var pRole *rbacv1.Role
		Eventually(func() error {
			pRole = &rbacv1.Role{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns1", Name: "role2"}, pRole)
		}).Should(Succeed())
		Expect(pRole.Labels).To(HaveKeyWithValue(constants.LabelCreatedBy, constants.CreatedBy))
		Expect(pRole.Annotations).To(HaveKeyWithValue(constants.AnnFrom, "tmpl"))
		Expect(pRole.Annotations).To(HaveKeyWithValue(constants.AnnPropagate, constants.PropagateCreate))
		Expect(pRole.Rules).To(HaveLen(1))

		var pSecret *corev1.Secret
		Eventually(func() error {
			pSecret = &corev1.Secret{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns1", Name: "foo"}, pSecret)
		}).Should(Succeed())
		Expect(pSecret.Labels).To(HaveKeyWithValue(constants.LabelCreatedBy, constants.CreatedBy))
		Expect(pSecret.Annotations).To(HaveKeyWithValue(constants.AnnFrom, "tmpl"))
		Expect(pSecret.Annotations).To(HaveKeyWithValue(constants.AnnPropagate, constants.PropagateUpdate))
		Expect(pSecret.Data).To(HaveKey("foo"))

		pRole2 := &rbacv1.Role{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns1", Name: "role1"}, pRole2)
		Expect(apierrors.IsNotFound(err)).To(BeTrue())

		pSecret2 := &corev1.Secret{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns1", Name: "bar"}, pSecret2)
		Expect(err).NotTo(HaveOccurred())

		By("changing a label of template namespace")
		tmpl = &corev1.Namespace{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "tmpl"}, tmpl)
		Expect(err).NotTo(HaveOccurred())
		tmpl.Labels["foo.bar/baz"] = "123"
		err = k8sClient.Update(ctx, tmpl)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			ns1 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "ns1"}, ns1); err != nil {
				return ""
			}
			return ns1.Labels["foo.bar/baz"]
		}).Should(Equal("123"))

		tmpl2 := &corev1.Namespace{}
		tmpl2.Name = "tmpl2"
		tmpl2.Labels = map[string]string{
			constants.LabelType: constants.NSTypeTemplate,
			"team":              "maneki",
		}
		err = k8sClient.Create(ctx, tmpl2)
		Expect(err).NotTo(HaveOccurred())

		sec2 := &corev1.Secret{}
		sec2.Namespace = "tmpl2"
		sec2.Name = "sec2"
		sec2.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateUpdate}
		sec2.Data = map[string][]byte{"foo": []byte("barbar")}
		err = k8sClient.Create(ctx, sec2)
		Expect(err).NotTo(HaveOccurred())

		By("changing the template namespace to tmpl2")
		ns1.Labels[constants.LabelTemplate] = "tmpl2"
		err = k8sClient.Update(ctx, ns1)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() bool {
			secret := &corev1.Secret{}
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns1", Name: "foo"}, secret)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())

		Eventually(func() error {
			secret := &corev1.Secret{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns1", Name: "sec2"}, secret)
		}).Should(Succeed())

		Eventually(func() string {
			ns1 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "ns1"}, ns1); err != nil {
				return ""
			}
			return ns1.Labels["team"]
		}).Should(Equal("maneki"))

		By("unsetting the template")
		ns1 = &corev1.Namespace{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "ns1"}, ns1)
		Expect(err).NotTo(HaveOccurred())

		delete(ns1.Labels, constants.LabelTemplate)
		err = k8sClient.Update(ctx, ns1)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() bool {
			secret := &corev1.Secret{}
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns1", Name: "sec2"}, secret)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())
	})

	It("should handle propagation between template namespaces", func() {
		tmpl1 := &corev1.Namespace{}
		tmpl1.Name = "tree-tmpl-1"
		tmpl1.Labels = map[string]string{
			constants.LabelType: constants.NSTypeTemplate,
			"team":              "neco",
		}
		err := k8sClient.Create(ctx, tmpl1)
		Expect(err).NotTo(HaveOccurred())

		tmpl2 := &corev1.Namespace{}
		tmpl2.Name = "tree-tmpl-2"
		tmpl2.Labels = map[string]string{
			constants.LabelType:     constants.NSTypeTemplate,
			constants.LabelTemplate: "tree-tmpl-1",
		}
		tmpl2.Annotations = map[string]string{"memo": "mome"}
		err = k8sClient.Create(ctx, tmpl2)
		Expect(err).NotTo(HaveOccurred())

		instance := &corev1.Namespace{}
		instance.Name = "tree-instance"
		instance.Labels = map[string]string{
			constants.LabelTemplate: "tree-tmpl-2",
		}
		err = k8sClient.Create(ctx, instance)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			instance = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "tree-instance"}, instance); err != nil {
				return ""
			}
			return instance.Labels["team"] + instance.Annotations["memo"]
		}).Should(Equal("necomome"))

		tmpl2 = &corev1.Namespace{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "tree-tmpl-2"}, tmpl2)
		Expect(err).NotTo(HaveOccurred())
		tmpl2.Labels["team"] = "hoge"
		tmpl2.Annotations["memo"] = "test"
		err = k8sClient.Update(ctx, tmpl2)
		Expect(err).NotTo(HaveOccurred())

		Consistently(func() string {
			instance = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "tree-instance"}, instance); err != nil {
				return ""
			}
			return instance.Labels["team"]
		}).Should(Equal("neco"))

		Expect(instance.Annotations["memo"]).Should(Equal("test"))

		tmpl2 = &corev1.Namespace{}
		err = k8sClient.Get(ctx, client.ObjectKey{Name: "tree-tmpl-2"}, tmpl2)
		Expect(err).NotTo(HaveOccurred())
		Expect(tmpl2.Labels["team"]).Should(Equal("neco"))
	})

	It("should implement a sub namespace correctly", func() {
		root := &corev1.Namespace{}
		root.Name = "root"
		root.Labels = map[string]string{
			constants.LabelType: constants.NSTypeRoot,
			"team":              "neco",
		}
		root.Annotations = map[string]string{
			"foo": "bar",
		}
		err := k8sClient.Create(ctx, root)
		Expect(err).NotTo(HaveOccurred())

		sec1 := &corev1.Secret{}
		sec1.Namespace = "root"
		sec1.Name = "sec1"
		sec1.Data = map[string][]byte{"foo": []byte("bar")}
		err = k8sClient.Create(ctx, sec1)
		Expect(err).NotTo(HaveOccurred())

		sec2 := &corev1.Secret{}
		sec2.Namespace = "root"
		sec2.Name = "sec2"
		sec2.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateCreate}
		sec2.Data = map[string][]byte{"foo": []byte("bar")}
		err = k8sClient.Create(ctx, sec2)
		Expect(err).NotTo(HaveOccurred())

		sec3 := &corev1.Secret{}
		sec3.Namespace = "root"
		sec3.Name = "sec3"
		sec3.Annotations = map[string]string{constants.AnnPropagate: constants.PropagateUpdate}
		sec3.Data = map[string][]byte{"foo": []byte("bar")}
		err = k8sClient.Create(ctx, sec3)
		Expect(err).NotTo(HaveOccurred())

		By("creating a sub namespace")
		sub1 := &corev1.Namespace{}
		sub1.Name = "sub1"
		sub1.Labels = map[string]string{constants.LabelParent: "root"}
		err = k8sClient.Create(ctx, sub1)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			sub1 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "sub1"}, sub1); err != nil {
				return ""
			}
			return sub1.Labels["team"]
		}).Should(Equal("neco"))
		Expect(sub1.Labels).NotTo(HaveKey(constants.LabelType))
		Expect(sub1.Annotations).NotTo(HaveKey("foo"))

		Eventually(func() error {
			cSec2 := &corev1.Secret{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "sub1", Name: "sec2"}, cSec2)
		}).Should(Succeed())
		Eventually(func() error {
			cSec3 := &corev1.Secret{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "sub1", Name: "sec3"}, cSec3)
		}).Should(Succeed())

		By("creating a grandchild namespace")
		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2"
		sub2.Labels = map[string]string{constants.LabelParent: "sub1"}
		err = k8sClient.Create(ctx, sub2)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			sub2 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "sub2"}, sub2); err != nil {
				return ""
			}
			return sub2.Labels["team"]
		}).Should(Equal("neco"))
		Eventually(func() error {
			cSec2 := &corev1.Secret{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "sub2", Name: "sec2"}, cSec2)
		}).Should(Succeed())
		Eventually(func() error {
			cSec3 := &corev1.Secret{}
			return k8sClient.Get(ctx, client.ObjectKey{Namespace: "sub2", Name: "sec3"}, cSec3)
		}).Should(Succeed())

		By("editing a label of root namespace")
		root.Labels["team"] = "nuco"
		err = k8sClient.Update(ctx, root)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			sub1 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "sub1"}, sub1); err != nil {
				return ""
			}
			return sub1.Labels["team"]
		}).Should(Equal("nuco"))
		Eventually(func() string {
			sub2 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "sub2"}, sub2); err != nil {
				return ""
			}
			return sub2.Labels["team"]
		}).Should(Equal("nuco"))

		By("changing the parent of sub2")
		root2 := &corev1.Namespace{}
		root2.Name = "root2"
		root2.Labels = map[string]string{
			constants.LabelType: constants.NSTypeRoot,
			"foo.bar/baz":       "baz",
		}
		err = k8sClient.Create(ctx, root2)
		Expect(err).NotTo(HaveOccurred())

		sub2.Labels[constants.LabelParent] = "root2"
		err = k8sClient.Update(ctx, sub2)
		Expect(err).NotTo(HaveOccurred())

		Eventually(func() string {
			sub2 = &corev1.Namespace{}
			if err := k8sClient.Get(ctx, client.ObjectKey{Name: "sub2"}, sub2); err != nil {
				return ""
			}
			return sub2.Labels["foo.bar/baz"]
		}).Should(Equal("baz"))

		Eventually(func() bool {
			sec := &corev1.Secret{}
			err := k8sClient.Get(ctx, client.ObjectKey{Namespace: "sub2", Name: "sec3"}, sec)
			return apierrors.IsNotFound(err)
		}).Should(BeTrue())

		cSec2 := &corev1.Secret{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "sub2", Name: "sec2"}, cSec2)
		Expect(err).NotTo(HaveOccurred())
	})
})
