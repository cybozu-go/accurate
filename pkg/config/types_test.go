package config

import (
	_ "embed"
	"testing"

	"github.com/google/go-cmp/cmp"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery/cached/memory"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/restmapper"
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

	if len(c.NamingPolicies) != 2 {
		t.Error("wrong number of namingPolicies:", len(c.NamingPolicies))
	}

	c = &Config{}
	err = c.Load(invalidData)
	if err == nil {
		t.Fatal("invalid data are loaded successfully")
	}
	t.Log(err)
}

func TestValidate(t *testing.T) {
	m := newFakeRESTMapper()
	testcases := []struct {
		config  *Config
		isValid bool
	}{
		{
			config: &Config{
				LabelKeys:      []string{"a", "b"},
				AnnotationKeys: []string{"foo", "bar"},
				Watches: []metav1.GroupVersionKind{
					{
						Group:   "",
						Version: "v1",
						Kind:    "Secret",
					},
					{
						Group:   "apps",
						Version: "v1",
						Kind:    "Deployment",
					},
				},
				NamingPolicies: []NamingPolicy{
					{
						Root:  "foo",
						Match: "bar",
					},
					{
						Root:  "a",
						Match: "b",
					},
				},
			},
			isValid: true,
		},
		{
			config: &Config{
				NamingPolicies: []NamingPolicy{
					{
						Root:  "(",
						Match: "abc",
					},
				},
			},
			isValid: false,
		},
		{
			config: &Config{
				NamingPolicies: []NamingPolicy{
					{
						Root:  "abc",
						Match: "(",
					},
				},
			},
			isValid: false,
		},
	}

	for _, testcase := range testcases {
		err := testcase.config.Validate(m)
		if testcase.isValid && err != nil {
			t.Fatal(err)
		}
		if !testcase.isValid && err == nil {
			t.Fatal("invalid data are validated successfully")
		}
	}
}

func newFakeRESTMapper() meta.RESTMapper {
	cs := &fakeclientset.Clientset{}
	cs.Fake.Resources = append(cs.Fake.Resources, &metav1.APIResourceList{
		GroupVersion: "v1",
		APIResources: []metav1.APIResource{
			{Name: "secrets", Namespaced: true, Kind: "Secret"},
		},
	}, &metav1.APIResourceList{
		GroupVersion: "apps/v1",
		APIResources: []metav1.APIResource{
			{Name: "deployments", Namespaced: true, Kind: "Deployment"},
		},
	})
	fakeDiscovery := &fakediscovery.FakeDiscovery{Fake: &cs.Fake}
	return restmapper.NewDeferredDiscoveryRESTMapper(memory.NewMemCacheClient(fakeDiscovery))
}
