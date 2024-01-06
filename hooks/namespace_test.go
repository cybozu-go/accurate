package hooks

import (
	"context"

	"github.com/cybozu-go/accurate/pkg/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Namespace webhook", func() {
	ctx := context.Background()

	It("should allow creating a normal namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "normal"
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		By("referencing a non-template as a template")
		ns.Labels = map[string]string{constants.LabelTemplate: "default"}
		err := k8sClient.Update(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should allow referencing a template namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "tmpl1"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		instance := &corev1.Namespace{}
		instance.Name = "instance-of-tmpl1"
		instance.Labels = map[string]string{
			constants.LabelTemplate: "tmpl1",
		}
		Expect(k8sClient.Create(ctx, instance)).To(Succeed())

		By("removing accurate.cybozu.com/type label from tmpl1")
		ns.Labels = nil
		err := k8sClient.Update(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should deny creating a self-referencing namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "self-reference"
		ns.Labels = map[string]string{
			constants.LabelParent: "self-reference",
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).To(HaveOccurred())

		ns.Labels = map[string]string{
			constants.LabelType:     constants.NSTypeTemplate,
			constants.LabelTemplate: "self-reference",
		}
		err = k8sClient.Create(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should deny creating a sub-namespace having a template", func() {
		ns := &corev1.Namespace{}
		ns.Name = "template-sub"
		ns.Labels = map[string]string{
			constants.LabelParent:   "kube-system",
			constants.LabelTemplate: "default",
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should deny creating a dangling sub-namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "create-dangling"
		ns.Labels = map[string]string{constants.LabelParent: "notexist"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should deny creating a sub-namespace under non-root/non-sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "non-root-non-sub"
		ns.Labels = map[string]string{constants.LabelType: "not-a-root"}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-non-root-non-sub"
		sub.Labels = map[string]string{constants.LabelParent: "non-root-non-sub"}
		err := k8sClient.Create(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should allow creating a sub-namespace under a root namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "create-root"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-create-root"
		sub.Labels = map[string]string{constants.LabelParent: "create-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())
	})

	It("should allow creating a sub-namespace under another sub-namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "create-root2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-create-root2"
		sub.Labels = map[string]string{constants.LabelParent: "create-root2"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-create-root2"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-create-root2"}
		Expect(k8sClient.Create(ctx, sub2)).To(Succeed())
	})

	It("should deny updating a sub-namespace that would create a circular reference", func() {
		ns := &corev1.Namespace{}
		ns.Name = "non-circular-root"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-non-circular-root"
		sub.Labels = map[string]string{constants.LabelParent: "non-circular-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-non-circular-root"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-non-circular-root"}
		Expect(k8sClient.Create(ctx, sub2)).To(Succeed())

		sub.Labels[constants.LabelParent] = "sub2-of-non-circular-root"
		err := k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny updating a template namespace that would create a circular reference", func() {
		ns := &corev1.Namespace{}
		ns.Name = "non-circular-root2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-non-circular-root2"
		sub.Labels = map[string]string{
			constants.LabelType:     constants.NSTypeTemplate,
			constants.LabelTemplate: "non-circular-root2",
		}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-non-circular-root2"
		sub2.Labels = map[string]string{
			constants.LabelType:     constants.NSTypeTemplate,
			constants.LabelTemplate: "sub-of-non-circular-root2",
		}
		Expect(k8sClient.Create(ctx, sub2)).To(Succeed())

		sub.Labels[constants.LabelTemplate] = "sub2-of-non-circular-root2"
		err := k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny updating a sub-namespace to have a template", func() {
		tmpl := &corev1.Namespace{}
		tmpl.Name = "dusht-tmpl"
		tmpl.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, tmpl)).To(Succeed())

		ns := &corev1.Namespace{}
		ns.Name = "template-root"
		ns.Labels = map[string]string{
			constants.LabelType:     constants.NSTypeRoot,
			constants.LabelTemplate: "dusht-tmpl",
		}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-template-root"
		sub.Labels = map[string]string{constants.LabelParent: "template-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub.Labels[constants.LabelTemplate] = "dusht-tmpl"
		err := k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny marking a sub-namespace as a root namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-of-sub-mark"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-root-of-sub-mark"
		sub.Labels = map[string]string{constants.LabelParent: "root-of-sub-mark"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub.Labels[constants.LabelType] = constants.NSTypeRoot
		err := k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny updating a namespace having children that would become a non-root and non-sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-after-non-root"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-root-after-non-root"
		sub.Labels = map[string]string{constants.LabelParent: "root-after-non-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-root-after-non-root"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-root-after-non-root"}
		Expect(k8sClient.Create(ctx, sub2)).To(Succeed())

		delete(ns.Labels, constants.LabelType)
		err := k8sClient.Update(ctx, ns)
		Expect(err).To(HaveOccurred())

		delete(sub.Labels, constants.LabelParent)
		err = k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should allow turning a root namespace into non-root if it has no children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-after-non-root3"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		ns.Labels = nil
		Expect(k8sClient.Update(ctx, ns)).To(Succeed())
	})

	It("should deny updating a template namespace having one or more instances that would become a non-template", func() {
		ns := &corev1.Namespace{}
		ns.Name = "tmpl-to-non-tmpl"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		child := &corev1.Namespace{}
		child.Name = "child-of-tmpl-to-non-tmpl"
		child.Labels = map[string]string{constants.LabelTemplate: "tmpl-to-non-tmpl"}
		Expect(k8sClient.Create(ctx, child)).To(Succeed())

		delete(ns.Labels, constants.LabelType)
		err := k8sClient.Update(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should allow turning a template w/o children namespace into a normal namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "tmpl-to-non-tmpl2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		delete(ns.Labels, constants.LabelType)
		Expect(k8sClient.Update(ctx, ns)).To(Succeed())
	})

	It("should allow turning a sub-namespace w/o children into a normal namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-of-depth1"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-root-of-depth1"
		sub.Labels = map[string]string{constants.LabelParent: "root-of-depth1"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub.Labels = nil
		Expect(k8sClient.Update(ctx, sub)).To(Succeed())
	})

	It("should allow turning a sub-namespace w/ children into a root namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-for-sub-to-root"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-root-for-sub-to-root"
		sub.Labels = map[string]string{constants.LabelParent: "root-for-sub-to-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-root-for-sub-to-root"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-root-for-sub-to-root"}
		Expect(k8sClient.Create(ctx, sub2)).To(Succeed())

		sub.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Update(ctx, sub)).To(Succeed())
	})

	It("should deny changing a sub-namespace into a dangling sub-namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "dangling-root"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-dangling-root"
		sub.Labels = map[string]string{constants.LabelParent: "dangling-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub.Labels[constants.LabelParent] = "none"
		err := k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny changing an instance namespace into a dangling namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "dangling-root2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-dangling-root2"
		sub.Labels = map[string]string{constants.LabelTemplate: "dangling-root2"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub.Labels[constants.LabelTemplate] = "none"
		err := k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny moving a sub-namespace under non-root/non-sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "move-root"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-move-root"
		sub.Labels = map[string]string{constants.LabelParent: "move-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		nonRoot := &corev1.Namespace{}
		nonRoot.Name = "move-non-root-non-sub"
		Expect(k8sClient.Create(ctx, nonRoot)).To(Succeed())

		sub.Labels[constants.LabelParent] = "move-non-root-non-sub"
		err := k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny deleting a root namespace w/ children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-delete-root"
		sub.Labels = map[string]string{constants.LabelParent: "delete-root"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		ns = &corev1.Namespace{}
		ns.Name = "delete-root"
		err := k8sClient.Delete(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should deny deleting a sub-namespace w/ children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-delete-root2"
		sub.Labels = map[string]string{constants.LabelParent: "delete-root2"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-delete-root2"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-delete-root2"}
		Expect(k8sClient.Create(ctx, sub2)).To(Succeed())

		sub = &corev1.Namespace{}
		sub.Name = "sub-of-delete-root2"
		err := k8sClient.Delete(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny deleting a template w/ children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-tmpl1"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-delete-tmpl1"
		sub.Labels = map[string]string{constants.LabelTemplate: "delete-tmpl1"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		err := k8sClient.Delete(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should allow deleting a sub-namespace w/o children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root3"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-delete-root3"
		sub.Labels = map[string]string{constants.LabelParent: "delete-root3"}
		Expect(k8sClient.Create(ctx, sub)).To(Succeed())

		sub = &corev1.Namespace{}
		sub.Name = "sub-of-delete-root3"
		Expect(k8sClient.Delete(ctx, sub)).To(Succeed())
	})

	It("should allow deleting a root namespace w/o children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root4"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		ns = &corev1.Namespace{}
		ns.Name = "delete-root4"
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	It("should allow deleting a non-root and non-sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root5"
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		ns = &corev1.Namespace{}
		ns.Name = "delete-root5"
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})

	It("should allow deleting a template namespace w/o instances", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-tmpl2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
		Expect(k8sClient.Create(ctx, ns)).To(Succeed())

		ns = &corev1.Namespace{}
		ns.Name = "delete-tmpl2"
		Expect(k8sClient.Delete(ctx, ns)).To(Succeed())
	})
})
