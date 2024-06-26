# Configurations

## Helm Chart values

Read [Helm Chart](helm.md) for details.

## Configuration file

[`accurate-controller`](accurate-controller.md) reads its configurations from a configuration file.

The repository includes an example as follows:

```yaml
# Labels to be propagated to sub-namespaces.
# It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
# https://pkg.go.dev/path#Match
labelKeys:
- team

# Annotations to be propagated to sub-namespaces.
# It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
# https://pkg.go.dev/path#Match
annotationKeys:
# An example to propagate an annotation for MetalLB
# https://metallb.universe.tf/usage/#requesting-specific-ips
- metallb.universe.tf/address-pool

# Labels to be propagated to sub-namespaces from SubNamespace resource.
# It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
# https://pkg.go.dev/path#Match
subNamespaceLabelKeys:
- app

# Annotations to be propagated to sub-namespaces from SubNamespace resource.
# It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
# https://pkg.go.dev/path#Match
subNamespaceAnnotationKeys:
- foo.bar/baz

# List of GVK for namespace-scoped resources that can be propagated.
# Any namespace-scoped resource is allowed.
watches:
- group: rbac.authorization.k8s.io
  version: v1
  kind: Role
- group: rbac.authorization.k8s.io
  version: v1
  kind: RoleBinding
- version: v1
  kind: Secret
- version: v1
  kind: ResourceQuota

# List of nameing policy for SubNamespaces.
# root and match are both regular expressions.
# When a SubNamespace is created in a tree starting from a root namespace and the root namespace's name matches the "root" regular expression, the SubNamespace name is validated with the "match" regular expression.
#
# "match" namingPolicies can use variables of regexp capture group naming of "root" namingPolicies.
# example:
#   root: ^app-(?P<team>.*)
#   match: ^app-${team}-.*
#   root namespace: app-team1
#   compiled match naming policy: ^app-team1-.*
# This feature is provided using https://pkg.go.dev/regexp#Regexp.Expand
namingPolicies: []
```

Only labels and annotations specified in the configuration file will be inherited.  
Be careful that some labels or annotations affect security configurations or the system.
For example, [`pod-security.kubernetes.io/*`](https://kubernetes.io/docs/concepts/security/pod-security-admission/#pod-security-admission-labels-for-namespaces) labels control the security capabilities of Pods in a Namespace.

Likewise, Accurate watches only namespace-scope resources specified in the configuration file.

You can edit the Helm Chart values as needed.

```yaml
<snip>
controller:
  config:
    # controller.config.labelKeys -- Labels to be propagated to sub-namespaces.
    # It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
    ## https://pkg.go.dev/path#Match
    labelKeys: []
    # - team

    # controller.config.annotationKeys -- Annotations to be propagated to sub-namespaces.
    # It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
    ## https://pkg.go.dev/path#Match
    annotationKeys: []
    # An example to propagate an annotation for MetalLB
    # https://metallb.universe.tf/usage/#requesting-specific-ips
    # - metallb.universe.tf/address-pool

    # Labels to be propagated to sub-namespaces from SubNamespace resource.
    # It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
    # https://pkg.go.dev/path#Match
    subNamespaceLabelKeys:
    - app

    # Annotations to be propagated to sub-namespaces from SubNamespace resource.
    # It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
    # https://pkg.go.dev/path#Match
    subNamespaceAnnotationKeys:
    - foo.bar/baz

    # controller.config.watches -- List of GVK for namespace-scoped resources that can be propagated.
    # Any namespace-scoped resource is allowed.
    watches:
      - group: rbac.authorization.k8s.io
        version: v1
        kind: Role
      - group: rbac.authorization.k8s.io
        version: v1
        kind: RoleBinding
      - version: v1
        kind: Secret

    # controller.config.namingPolicies -- List of nameing policy for SubNamespaces.
    # root and match are both regular expressions.
    # When a SubNamespace is created in a tree starting from a root namespace and the root namespace's name matches the "root" regular expression, the SubNamespace name is validated with the "match" regular expression.
    #
    # "match" namingPolicies can use variables of regexp capture group naming of "root" namingPolicies.
    # example:
    #   root: ^app-(?P<team>.*)
    #   match: ^app-${team}-.*
    #   root namespace: app-team1
    #   compiled match naming policy: ^app-team1-.*
    # This feature is provided using https://pkg.go.dev/regexp#Regexp.Expand
    namingPolicies:
      - root:  foo
        match: foo_.*
      - root:  bar
        match: bar_.*
      - root:  ^app-(?P<team>.*)
        match: ^app-${team}-.*
<snip>
```

## ClusterRoleBindings

A built-in ClusterRole `admin` is bound by default to allow `accurate-controller` to watch and propagate namespace-scope resources. However, `admin` does not contain verbs for [ResourceQuota][] and may not contain custom resources.

If you need to watch and propagate resources not included in `admin` ClusterRole, add additional ClusterRole/ClusterRoleBinding to `accurate-controller-manager` ServiceAccount.
Set the `controller.additionalRBAC.rules` in the Helm Chart values.

The following example Helm chart values is to watch and propagate ResourceQuotas.

```yaml
<snip>
controller:
  additionalRBAC:
    # controller.additionalRBAC.rules -- Specify the RBAC rules to be added to the controller.
    # ClusterRole and ClusterRoleBinding are created with the names `{{ release name }}-additional-resources`.
    # The rules defined here will be used for the ClusterRole rules.
    rules:
      - apiGroups:
          - ""
        resources:
          - resourcequotas
        verbs:
          - get
          - list
          - watch
          - create
          - patch
          - delete
<snip>
```

## Feature Gates

Feature gates are a set of key=value pairs that describe operator features.
You can turn these features on or off using the `--feature-gates` command line flag.
Use `-h` flag to see a full set of feature gates.

To set feature gates, use the `--feature-gates` flag assigned to a list of feature pairs:

```shell
--feature-gates=...,DisablePropagateGenerated=false
```

The following table is a summary of the feature gates that you can set.

- The "Since" column contains the Accurate release when a feature is introduced
  or its release stage is changed.
- The "Until" column, if not empty, contains the last Accurate release in which
  you can still use a feature gate.

{{< table caption="Feature gates for features in Alpha or Beta states" sortable="true" >}}

| Feature | Default | Stage | Since | Until |
|---------|---------|-------|-------|-------|
| `DisablePropagateGenerated` | `false` | Alpha | 1.2.0 | 1.3.0 |
| `DisablePropagateGenerated` | `true` | Beta | 1.3.0 | |

Each feature gate is designed for enabling/disabling a specific feature:

- `DisablePropagateGenerated`: Disable [propagating generated resources](concepts.md#propagating-generated-resources),
  which is a feature subject for removal soon.

[ResourceQuota]: https://kubernetes.io/docs/concepts/policy/resource-quotas/
