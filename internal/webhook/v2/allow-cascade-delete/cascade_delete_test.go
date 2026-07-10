package hooks_allow_cascade_delete

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	"github.com/cybozu-go/accurate/pkg/constants"
)

var _ = Describe("Webhook allow cascade delete", func() {
	var (
		ctx  context.Context
		root *corev1.Namespace
	)

	BeforeEach(func() {
		ctx = context.Background()

		root = &corev1.Namespace{}
		root.GenerateName = "cascade-root-"
		root.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		Expect(k8sClient.Create(ctx, root)).To(Succeed())
	})

	Context("Namespace", func() {
		It("should ALLOW deleting a root with children", func() {
			sub := &corev1.Namespace{}
			sub.GenerateName = "cascade-sub-"
			sub.Labels = map[string]string{constants.LabelParent: root.Name}
			Expect(k8sClient.Create(ctx, sub)).To(Succeed())

			Expect(k8sClient.Delete(ctx, root)).To(Succeed())
		})

		It("should ALLOW deleting a sub-namespace with children", func() {
			sub := &corev1.Namespace{}
			sub.GenerateName = "cascade-sub-"
			sub.Labels = map[string]string{constants.LabelParent: root.Name}
			Expect(k8sClient.Create(ctx, sub)).To(Succeed())

			subSub := &corev1.Namespace{}
			subSub.GenerateName = "cascade-sub-sub-"
			subSub.Labels = map[string]string{constants.LabelParent: sub.Name}
			Expect(k8sClient.Create(ctx, subSub)).To(Succeed())

			Expect(k8sClient.Delete(ctx, sub)).To(Succeed())
		})

		It("should DENY deleting a template with children", func() {
			tmpl := &corev1.Namespace{}
			tmpl.GenerateName = "cascade-tmpl-"
			tmpl.Labels = map[string]string{constants.LabelType: constants.NSTypeTemplate}
			Expect(k8sClient.Create(ctx, tmpl)).To(Succeed())

			root.Labels[constants.LabelTemplate] = tmpl.Name
			Expect(k8sClient.Update(ctx, root)).To(Succeed())

			err := k8sClient.Delete(ctx, tmpl)
			Expect(err).To(HaveOccurred())
			Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonForbidden))
		})
	})

	Context("SubNamespace", func() {
		It("should ALLOW deletion with child namespaces", func() {
			sub := &accuratev2.SubNamespace{}
			sub.Namespace = root.Name
			sub.GenerateName = "cascade-sub-"
			Expect(k8sClient.Create(ctx, sub)).To(Succeed())
			// Create sub-namespace since no controllers present in this test setup
			subNS := &corev1.Namespace{}
			subNS.Name = sub.Name
			subNS.Labels = map[string]string{constants.LabelParent: root.Name}
			Expect(k8sClient.Create(ctx, subNS)).To(Succeed())

			ns := &corev1.Namespace{}
			ns.GenerateName = "cascade-sub-sub-"
			ns.Labels = map[string]string{constants.LabelParent: subNS.Name}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			Expect(k8sClient.Delete(ctx, sub)).To(Succeed())
		})
	})
})
