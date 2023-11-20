package hooks

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

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

	It("should deny creation of v2 SubNamespace in a namespace that is neither root nor subnamespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "v2alpha1-ns1"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sn := &accuratev2alpha1.SubNamespace{}
		sn.Namespace = "v2alpha1-ns1"
		sn.Name = "v2alpha1-foo"
		err = k8sClient.Create(ctx, sn)
		Expect(err).To(HaveOccurred())
		Expect(errors.ReasonForError(err)).Should(Equal(metav1.StatusReasonForbidden))
	})

	It("should allow creation of SubNamespace in a root namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "v2alpha1-ns2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		err := k8sClient.Create(ctx, ns)
		Expect(err).To(Succeed())

		sn := &accuratev2alpha1.SubNamespace{}
		sn.Namespace = "v2alpha1-ns2"
		sn.Name = "v2alpha1-foo"
		sn.Spec.Labels = map[string]string{"foo": "bar"}
		sn.Spec.Annotations = map[string]string{"foo": "bar"}
		err = k8sClient.Create(ctx, sn)
		Expect(err).To(Succeed())

		Expect(controllerutil.ContainsFinalizer(sn, constants.Finalizer)).To(BeTrue())

		// deleting finalizer should succeed
		sn.Finalizers = nil
		err = k8sClient.Update(ctx, sn)
		Expect(err).To(Succeed())

		sn = &accuratev2alpha1.SubNamespace{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "v2alpha1-ns2", Name: "v2alpha1-foo"}, sn)
		Expect(err).To(Succeed())
		Expect(sn.Finalizers).To(BeEmpty())
	})

})
