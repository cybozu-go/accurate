package config

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/cybozu-go/accurate/pkg/constants"
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
	LabelKeys                      []string                  `json:"labelKeys,omitempty"`
	AnnotationKeys                 []string                  `json:"annotationKeys,omitempty"`
	SubNamespaceLabelKeys          []string                  `json:"subNamespaceLabelKeys,omitempty"`
	SubNamespaceAnnotationKeys     []string                  `json:"subNamespaceAnnotationKeys,omitempty"`
	Watches                        []metav1.GroupVersionKind `json:"watches,omitempty"`
	PropagateLabelKeyExcludes      []string                  `json:"propagateLabelKeyExcludes,omitempty"`
	PropagateAnnotationKeyExcludes []string                  `json:"propagateAnnotationKeyExcludes,omitempty"`
	NamingPolicies                 []NamingPolicy            `json:"namingPolicies,omitempty"`
	NamingPolicyRegexps            []NamingPolicyRegexp
}

// Validate validates the configurations.
func (c *Config) Validate(mapper meta.RESTMapper) error {
	for _, key := range c.LabelKeys {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for labelKeys %s: %w", key, err)
		}
		if strings.HasPrefix(key, constants.MetaPrefix) {
			return fmt.Errorf("misconfigured labelKey: %s is not allowed", key)
		}
	}

	for _, key := range c.AnnotationKeys {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for annotationKeys %s: %w", key, err)
		}
		if strings.HasPrefix(key, constants.MetaPrefix) {
			return fmt.Errorf("misconfigured annotationKey: %s is not allowed", key)
		}
	}

	for _, key := range c.SubNamespaceLabelKeys {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for subNamespaceLabelKeys %s: %w", key, err)
		}
		if strings.HasPrefix(key, constants.MetaPrefix) {
			return fmt.Errorf("misconfigured subNamespaceLabelKey: %s is not allowed", key)
		}
	}

	for _, key := range c.SubNamespaceAnnotationKeys {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for subNamespaceAnnotationKeys %s: %w", key, err)
		}
		if strings.HasPrefix(key, constants.MetaPrefix) {
			return fmt.Errorf("misconfigured subNamespaceAnnotationKey: %s is not allowed", key)
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

	for _, key := range c.PropagateLabelKeyExcludes {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for propagateLabelKeyExcludes %s: %w", key, err)
		}
	}

	for _, key := range c.PropagateAnnotationKeyExcludes {
		// Verify that pattern is a valid format.
		if _, err := path.Match(key, ""); err != nil {
			return fmt.Errorf("malformed pattern for propagateAnnotationKeyExcludes %s: %w", key, err)
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

// ValidateRBAC validates that the manager has RBAC permissions to support configuration
func (c *Config) ValidateRBAC(ctx context.Context, client client.Client, mapper meta.RESTMapper) error {
	var errList []error

	for _, gvk := range c.Watches {
		mapping, err := mapper.RESTMapping(schema.GroupKind{Group: gvk.Group, Kind: gvk.Kind}, gvk.Version)
		if err != nil {
			return fmt.Errorf("error mapping GVK %s: %w", gvk.String(), err)
		}

		selfCheck := &authv1.SelfSubjectAccessReview{}
		selfCheck.Spec.ResourceAttributes = &authv1.ResourceAttributes{
			Group:    mapping.Resource.Group,
			Version:  mapping.Resource.Version,
			Resource: mapping.Resource.Resource,
		}
		for _, verb := range []string{"get", "list", "watch", "create", "patch", "delete"} {
			selfCheck.Spec.ResourceAttributes.Verb = verb
			if err := client.Create(ctx, selfCheck); err != nil {
				return fmt.Errorf("error creating SelfSubjectAccessReview: %w", err)
			}
			if !selfCheck.Status.Allowed {
				errList = append(errList, fmt.Errorf("missing permission to %s %s", verb, mapping.Resource.String()))
			}
		}
	}

	return errors.NewAggregate(errList)
}

// Load loads configurations.
func (c *Config) Load(data []byte) error {
	return yaml.Unmarshal(data, c, yaml.DisallowUnknownFields)
}
