//go:build envtest

package config

import (
	"context"
	_ "embed"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Validate", func() {
	It("should pass an empty config", func() {
		c := &Config{}
		Expect(c.Validate(mapper)).To(Succeed())
	})

	It("should pass any labels/annotations in config", func() {
		c := &Config{
			LabelKeys:      []string{"1", "2"},
			AnnotationKeys: []string{"a", "b"},
		}
		Expect(c.Validate(mapper)).To(Succeed())
	})

	It("should pass any SubNamespace labels/annotations in config", func() {
		c := &Config{
			SubNamespaceLabelKeys:      []string{"1", "2"},
			SubNamespaceAnnotationKeys: []string{"a", "b"},
		}
		Expect(c.Validate(mapper)).To(Succeed())
	})

	It("should deny labelKeys in accurate's own namespace", func() {
		c := &Config{
			LabelKeys: []string{"accurate.cybozu.com/type"},
		}
		Expect(c.Validate(mapper)).NotTo(Succeed())
	})

	It("should deny annotationKeys in accurate's own namespace", func() {
		c := &Config{
			AnnotationKeys: []string{"accurate.cybozu.com/type"},
		}
		Expect(c.Validate(mapper)).NotTo(Succeed())
	})

	It("should deny subNamespaceLabelKeys in accurate's own namespace", func() {
		c := &Config{
			SubNamespaceLabelKeys: []string{"accurate.cybozu.com/type"},
		}
		Expect(c.Validate(mapper)).NotTo(Succeed())
	})

	It("should deny subNamespaceAnnotationKeys in accurate's own namespace", func() {
		c := &Config{
			SubNamespaceAnnotationKeys: []string{"accurate.cybozu.com/type"},
		}
		Expect(c.Validate(mapper)).NotTo(Succeed())
	})

	It("should pass watches for namespace-scoped resources", func() {
		c := &Config{
			Watches: []metav1.GroupVersionKind{{
				Group:   "rbac.authorization.k8s.io",
				Version: "v1",
				Kind:    "Role",
			}},
		}
		Expect(c.Validate(mapper)).To(Succeed())
	})

	It("should deny cluster-scoped resources in watches", func() {
		c := &Config{
			Watches: []metav1.GroupVersionKind{{
				Group:   "rbac.authorization.k8s.io",
				Version: "v1",
				Kind:    "ClusterRole",
			}},
		}
		Expect(c.Validate(mapper)).NotTo(Succeed())
	})
})

var _ = Describe("ValidateRBAC", func() {
	var c *Config
	var ctx context.Context

	BeforeEach(func() {
		c = &Config{
			Watches: []metav1.GroupVersionKind{{
				Group:   "rbac.authorization.k8s.io",
				Version: "v1",
				Kind:    "Role",
			}},
		}
		ctx = context.Background()
	})

	It("should succeed when RBAC present to watched resources", func() {
		Expect(c.ValidateRBAC(ctx, fullAccessClient, mapper)).To(Succeed())
	})

	It("should error when missing RBAC to watched resources", func() {
		Expect(c.ValidateRBAC(ctx, noAccessClient, mapper)).To(MatchError(ContainSubstring("missing permission to patch rbac.authorization.k8s.io/v1, Resource=roles")))
	})
})
