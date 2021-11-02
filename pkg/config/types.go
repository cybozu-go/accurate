package config

import (
	"fmt"
	"path"
	"regexp"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/yaml"
)

// NamingPolicy represents naming policies for Namespaces created from SubNamespaces.
type NamingPolicy struct {
	Root  string `json:"root"`
	Match string `json:"match"`
}

type NamingPolicyRegexp struct {
	Root  *regexp.Regexp
	Match string
}

// Config represents the configuration file of Accurate.
type Config struct {
	LabelKeys           []string                  `json:"labelKeys,omitempty"`
	AnnotationKeys      []string                  `json:"annotationKeys,omitempty"`
	Watches             []metav1.GroupVersionKind `json:"watches,omitempty"`
	NamingPolicies      []NamingPolicy            `json:"namingPolicies,omitempty"`
	NamingPolicyRegexps []NamingPolicyRegexp
}

// Validate validates the configurations.
func (c *Config) Validate(mapper meta.RESTMapper) error {
	for _, key := range c.LabelKeys {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for labelKeys %s: %w", key, err)
		}
	}

	for _, key := range c.AnnotationKeys {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for annotationKeys %s: %w", key, err)
		}
	}

	for _, gvk := range c.Watches {
		mapping, err := mapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
		if err != nil {
			return fmt.Errorf("invalid gvk %s: %w", gvk.String(), err)
		}

		if mapping.Scope.Name() != meta.RESTScopeNameNamespace {
			return fmt.Errorf("%s is not namespace-scoped", gvk.String())
		}
	}

	for _, policy := range c.NamingPolicies {
		root, err := regexp.Compile(policy.Root)
		if err != nil {
			return fmt.Errorf("invalid naming policy: %w", err)
		}
		c.NamingPolicyRegexps = append(c.NamingPolicyRegexps, NamingPolicyRegexp{Root: root, Match: policy.Match})
	}
	return nil
}

// Load loads configurations.
func (c *Config) Load(data []byte) error {
	return yaml.Unmarshal(data, c, yaml.DisallowUnknownFields)
}
