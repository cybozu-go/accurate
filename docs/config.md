# Configurations

## Configuration file

[`accurate-controller`](accurate-controller.md) reads its configurations from a configuration file.

The repository includes an example as follows:

```yaml
{{#include ../config.yaml}}
```

Only labels and annotations specified in the configuration file will be inherited.

Likewise, Accurate watches only namespace-scope resources specified in the configuration file.

Edit `config.yaml` in the top directory of the repository.
The file will be embedded in ConfigMap laster using `kustomize`.

## ClusterRoleBindings

`config/rbac/role_binding.yaml` contains ClusterRoleBindings for `accurate-controller`.

A built-in ClusterRole `admin` is bound by default to allow `accurate-controller` to watch and propagate namespace-scope resources.

If you need to watch and propagate resources not included in `admin` ClusterRole, add additional ClusterRole/ClusterRoleBinding to `accurate-controller-manager` ServiceAccount in `accurate` namespace.
