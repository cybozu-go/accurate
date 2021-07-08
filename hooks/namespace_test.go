package hooks

import (
	"context"

	"github.com/cybozu-go/innu/pkg/constants"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

var _ = Describe("Namespace webhook", func() {
	ctx := context.Background()

	It("should allow creating a normal namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "normal"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		ns.Labels = map[string]string{
			constants.LabelTemplate: "default",
		}
		err = k8sClient.Update(ctx, ns)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should deny creating a self-referencing sub-namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "self-reference"
		ns.Labels = map[string]string{constants.LabelParent: "self-reference"}
		err := k8sClient.Create(ctx, ns)
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
		ns.Labels = map[string]string{constants.LabelRoot: "not-a-root"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-non-root-non-sub"
		sub.Labels = map[string]string{constants.LabelParent: "non-root-non-sub"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should allow creating a sub-namespace under a root namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "create-root"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-create-root"
		sub.Labels = map[string]string{constants.LabelParent: "create-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow creating a sub-namespace under another sub-namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "create-root2"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-create-root2"
		sub.Labels = map[string]string{constants.LabelParent: "create-root2"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-create-root2"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-create-root2"}
		err = k8sClient.Create(ctx, sub2)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should deny updating a sub-namespace that would create a circular reference", func() {
		ns := &corev1.Namespace{}
		ns.Name = "non-circular-root"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-non-circular-root"
		sub.Labels = map[string]string{constants.LabelParent: "non-circular-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-non-circular-root"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-non-circular-root"}
		err = k8sClient.Create(ctx, sub2)
		Expect(err).NotTo(HaveOccurred())

		sub.Labels[constants.LabelParent] = "sub2-of-non-circular-root"
		err = k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny updating a sub-namespace to have a template", func() {
		ns := &corev1.Namespace{}
		ns.Name = "template-root"
		ns.Labels = map[string]string{
			constants.LabelRoot:     "true",
			constants.LabelTemplate: "default",
		}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-template-root"
		sub.Labels = map[string]string{constants.LabelParent: "template-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub.Labels[constants.LabelTemplate] = "default"
		err = k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny updating a namespace having children that would become a non-root and non-sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-after-non-root"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-root-after-non-root"
		sub.Labels = map[string]string{constants.LabelParent: "root-after-non-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-root-after-non-root"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-root-after-non-root"}
		err = k8sClient.Create(ctx, sub2)
		Expect(err).NotTo(HaveOccurred())

		delete(ns.Labels, constants.LabelRoot)
		err = k8sClient.Update(ctx, ns)
		Expect(err).To(HaveOccurred())

		delete(sub.Labels, constants.LabelParent)
		err = k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should allow turning a root namespace into non-root if it has no children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-after-non-root3"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		ns.Labels = nil
		err = k8sClient.Update(ctx, ns)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow turning a sub-namespace w/o children into a normal namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-of-depth1"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-root-of-depth1"
		sub.Labels = map[string]string{constants.LabelParent: "root-of-depth1"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub.Labels = nil
		err = k8sClient.Update(ctx, sub)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow turning a sub-namespace w/ children into a root namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "root-for-sub-to-root"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-root-for-sub-to-root"
		sub.Labels = map[string]string{constants.LabelParent: "root-for-sub-to-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-root-for-sub-to-root"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-root-for-sub-to-root"}
		err = k8sClient.Create(ctx, sub2)
		Expect(err).NotTo(HaveOccurred())

		sub.Labels = map[string]string{constants.LabelRoot: "true"}
		err = k8sClient.Update(ctx, sub)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should deny changing a sub-namespace into a dangling sub-namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "dangling-root"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-dangling-root"
		sub.Labels = map[string]string{constants.LabelParent: "dangling-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub.Labels[constants.LabelParent] = "none"
		err = k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny moving a sub-namespace under non-root/non-sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "move-root"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-move-root"
		sub.Labels = map[string]string{constants.LabelParent: "move-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		nonRoot := &corev1.Namespace{}
		nonRoot.Name = "move-non-root-non-sub"
		err = k8sClient.Create(ctx, nonRoot)
		Expect(err).NotTo(HaveOccurred())

		sub.Labels[constants.LabelParent] = "move-non-root-non-sub"
		err = k8sClient.Update(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should deny deleting a root namespace w/ children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-delete-root"
		sub.Labels = map[string]string{constants.LabelParent: "delete-root"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		ns = &corev1.Namespace{}
		ns.Name = "delete-root"
		err = k8sClient.Delete(ctx, ns)
		Expect(err).To(HaveOccurred())
	})

	It("should deny deleting a sub-namespace w/ children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root2"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-delete-root2"
		sub.Labels = map[string]string{constants.LabelParent: "delete-root2"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub2 := &corev1.Namespace{}
		sub2.Name = "sub2-of-delete-root2"
		sub2.Labels = map[string]string{constants.LabelParent: "sub-of-delete-root2"}
		err = k8sClient.Create(ctx, sub2)
		Expect(err).NotTo(HaveOccurred())

		sub = &corev1.Namespace{}
		sub.Name = "sub-of-delete-root2"
		err = k8sClient.Delete(ctx, sub)
		Expect(err).To(HaveOccurred())
	})

	It("should allow deleting a sub-namespace w/o children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root3"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sub := &corev1.Namespace{}
		sub.Name = "sub-of-delete-root3"
		sub.Labels = map[string]string{constants.LabelParent: "delete-root3"}
		err = k8sClient.Create(ctx, sub)
		Expect(err).NotTo(HaveOccurred())

		sub = &corev1.Namespace{}
		sub.Name = "sub-of-delete-root3"
		err = k8sClient.Delete(ctx, sub)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow deleting a root namespace w/o children", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root4"
		ns.Labels = map[string]string{constants.LabelRoot: "true"}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		ns = &corev1.Namespace{}
		ns.Name = "delete-root4"
		err = k8sClient.Delete(ctx, ns)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should allow deleting a non-root and non-sub namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "delete-root5"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		ns = &corev1.Namespace{}
		ns.Name = "delete-root5"
		err = k8sClient.Delete(ctx, ns)
		Expect(err).NotTo(HaveOccurred())
	})
})
