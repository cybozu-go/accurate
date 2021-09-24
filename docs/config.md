# Configurations

## Helm Chart values

Read [Helm Chart](helm.md) for details.

## Configuration file

[`accurate-controller`](accurate-controller.md) reads its configurations from a configuration file.

The repository includes an example as follows:

```yaml
{{#include ../config.yaml}}
```

Only labels and annotations specified in the configuration file will be inherited.

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
<snip>
```

## ClusterRoleBindings

A built-in ClusterRole `admin` is bound by default to allow `accurate-controller` to watch and propagate namespace-scope resources.

If you need to watch and propagate resources not included in `admin` ClusterRole, add additional ClusterRole/ClusterRoleBinding to `accurate-controller-manager` ServiceAccount.
