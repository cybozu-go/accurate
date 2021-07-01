package config

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
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

//go:embed testdata/config.yaml
var validData []byte

//go:embed testdata/invalid.yaml
var invalidData []byte

func TestLoad(t *testing.T) {
	c := &Config{}
	err := c.Load(validData)
	if err != nil {
		t.Fatal(err)
	}

	if !cmp.Equal(c.LabelKeys, []string{"a", "b"}) {
		t.Error("wrong label keys:", cmp.Diff(c.LabelKeys, []string{"a", "b"}))
	}
	if !cmp.Equal(c.AnnotationKeys, []string{"foo", "bar"}) {
		t.Error("wrong annotation keys:", cmp.Diff(c.AnnotationKeys, []string{"foo", "bar"}))
	}

	if len(c.Watches) != 2 {
		t.Error("wrong number of watches:", len(c.Watches))
	}
	gvk := c.Watches[0]
	if gvk.Group != "apps" {
		t.Error("wrong group:", gvk.Group)
	}
	if gvk.Version != "v1" {
		t.Error("wrong version:", gvk.Version)
	}
	if gvk.Kind != "Deployment" {
		t.Error("wrong kind:", gvk.Kind)
	}

	c = &Config{}
	err = c.Load(invalidData)
	if err == nil {
		t.Fatal("invalid data are loaded successfully")
	}
	t.Log(err)
}
