package hooks

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	accuratev2 "github.com/cybozu-go/accurate/api/accurate/v2"
	accuratev2alpha1 "github.com/cybozu-go/accurate/api/accurate/v2alpha1"
	"github.com/cybozu-go/accurate/pkg/constants"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// This is just simple smoke tests ensuring our webhooks are handling additional versions of the SubNamespace API.
var _ = Describe("SubNamespace webhook", func() {
	ctx := context.Background()

	Context("v2alpha1", func() {
		It("should deny creation of SubNamespace in a namespace that is neither root nor subnamespace", func() {
			ns := &corev1.Namespace{}
			ns.Name = "v2alpha1-ns1"
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			sn := &accuratev2alpha1.SubNamespace{}
			sn.Namespace = ns.Name
			sn.Name = "v2alpha1-foo"
			err := k8sClient.Create(ctx, sn)
			Expect(err).To(HaveOccurred())
			Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonForbidden))
		})

		It("should allow creation of SubNamespace in a root namespace", func() {
			ns := &corev1.Namespace{}
			ns.Name = "v2alpha1-ns2"
			ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			sn := &accuratev2alpha1.SubNamespace{}
			sn.Namespace = ns.Name
			sn.Name = "v2alpha1-foo"
			sn.Spec.Labels = map[string]string{"foo": "bar"}
			sn.Spec.Annotations = map[string]string{"foo": "bar"}
			Expect(k8sClient.Create(ctx, sn)).To(Succeed())

			Expect(controllerutil.ContainsFinalizer(sn, constants.Finalizer)).To(BeTrue())

			// deleting finalizer should succeed
			sn.Finalizers = nil
			Expect(k8sClient.Update(ctx, sn)).To(Succeed())

			sn = &accuratev2alpha1.SubNamespace{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: "v2alpha1-ns2", Name: "v2alpha1-foo"}, sn)).To(Succeed())
			Expect(sn.Finalizers).To(BeEmpty())
		})
	})

	Context("v2", func() {
		It("should deny creation of SubNamespace in a namespace that is neither root nor subnamespace", func() {
			ns := &corev1.Namespace{}
			ns.Name = "v2-ns1"
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			sn := &accuratev2.SubNamespace{}
			sn.Namespace = ns.Name
			sn.Name = "v2-foo"
			err := k8sClient.Create(ctx, sn)
			Expect(err).To(HaveOccurred())
			Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonForbidden))
		})

		It("should allow creation of SubNamespace in a root namespace", func() {
			ns := &corev1.Namespace{}
			ns.Name = "v2-ns2"
			ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
			Expect(k8sClient.Create(ctx, ns)).To(Succeed())

			sn := &accuratev2.SubNamespace{}
			sn.Namespace = ns.Name
			sn.Name = "v2-foo"
			sn.Spec.Labels = map[string]string{"foo": "bar"}
			sn.Spec.Annotations = map[string]string{"foo": "bar"}
			Expect(k8sClient.Create(ctx, sn)).To(Succeed())

			Expect(controllerutil.ContainsFinalizer(sn, constants.Finalizer)).To(BeTrue())

			// deleting finalizer should succeed
			sn.Finalizers = nil
			Expect(k8sClient.Update(ctx, sn)).To(Succeed())

			sn = &accuratev2.SubNamespace{}
			Expect(k8sClient.Get(ctx, client.ObjectKey{Namespace: "v2-ns2", Name: "v2-foo"}, sn)).To(Succeed())
			Expect(sn.Finalizers).To(BeEmpty())
		})
	})
})
