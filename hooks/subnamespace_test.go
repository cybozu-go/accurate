package hooks

import (
	"context"

	innuv1 "github.com/cybozu-go/innu/api/v1"
	"github.com/cybozu-go/innu/pkg/constants"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

var _ = Describe("SubNamespace webhook", func() {
	ctx := context.Background()

	It("should deny creation of SubNamespace in a namespace that is neither root nor subnamespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "ns1"
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sn := &innuv1.SubNamespace{}
		sn.Namespace = "ns1"
		sn.Name = "foo"
		err = k8sClient.Create(ctx, sn)
		Expect(err).To(HaveOccurred())
	})

	It("should allow creation of SubNamespace in a root namespace", func() {
		ns := &corev1.Namespace{}
		ns.Name = "ns2"
		ns.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		err := k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sn := &innuv1.SubNamespace{}
		sn.Namespace = "ns2"
		sn.Name = "foo"
		err = k8sClient.Create(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		Expect(controllerutil.ContainsFinalizer(sn, constants.Finalizer)).To(BeTrue())

		// deleting finalizer should succeeds
		sn.Finalizers = nil
		err = k8sClient.Update(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		sn = &innuv1.SubNamespace{}
		err = k8sClient.Get(ctx, client.ObjectKey{Namespace: "ns2", Name: "foo"}, sn)
		Expect(err).NotTo(HaveOccurred())
		Expect(sn.Finalizers).To(BeEmpty())
	})

	It("should allow creation of SubNamespace in a subnamespace", func() {
		nsP := &corev1.Namespace{}
		nsP.Name = "ns-parent"
		nsP.Labels = map[string]string{constants.LabelType: constants.NSTypeRoot}
		err := k8sClient.Create(ctx, nsP)
		Expect(err).NotTo(HaveOccurred())

		ns := &corev1.Namespace{}
		ns.Name = "ns3"
		ns.Labels = map[string]string{constants.LabelParent: "ns-parent"}
		err = k8sClient.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred())

		sn := &innuv1.SubNamespace{}
		sn.Namespace = "ns3"
		sn.Name = "bar"
		err = k8sClient.Create(ctx, sn)
		Expect(err).NotTo(HaveOccurred())

		Expect(controllerutil.ContainsFinalizer(sn, constants.Finalizer)).To(BeTrue())
	})
})
