# Accurate Helm Chart

## How to use Accurate Helm repository

You need to add this repository to your Helm repositories:

```bash
helm repo add accurate https://cybozu-go.github.io/accurate/
helm repo update
```

## Quick start

### Installing cert-manager

```bash
curl -fsL https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml | kubectl apply -f -
```

### Installing CustomResourceDefinitions (optional)

Accurate does not use the [official helm method](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/) of installing CRD resources.
This is because it makes upgrading CRDs impossible with helm CLI alone.
The helm team explain the limitations of their approach [here](https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#some-caveats-and-explanations).

The Accurate Helm chart default is to install and manage CRDs with Helm and
add annotations preventing Helm from uninstalling the CRD when the Helm release is uninstalled.

The recommended approach is to let helm manage CRDs, but if you want to manage CRDs yourself, now is the time.

```bash
kubectl apply -k https://github.com/cybozu-go/accurate//config/crd-only/
```

> [!NOTE]
> Since the CRDs contain configuration of conversion webhooks, you may have to tweak the webhook settings
> if installing the chart using non-standard values.

If you decided to manage CRDs outside of Helm, make sure you set the `crds.enabled` Helm value to `false`.

### Installing the Chart

> [!NOTE]
> This installation method requires cert-manager to be installed beforehand.

To install the chart with the release name `accurate` using a dedicated namespace(recommended):

```bash
helm install --create-namespace --namespace accurate accurate accurate/accurate
```

Specify parameters using `--set key=value[,key=value]` argument to `helm install`.

Alternatively a YAML file that specifies the values for the parameters can be provided like this:

```bash
helm install --create-namespace --namespace accurate accurate -f values.yaml accurate/accurate
```

## Values

| Key                                              | Type   | Default                                                                                                                                                                           | Description                                                                                                                                                                                                                   |
|--------------------------------------------------|--------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| controller.additionalRBAC.rules                  | list   | `[]`                                                                                                                                                                              | Specify the RBAC rules to be added to the controller. ClusterRole and ClusterRoleBinding are created with the names `{{ release name }}-additional-resources`. The rules defined here will be used for the ClusterRole rules. |
| controller.additionalRBAC.clusterRoles           | list   | `[]`                                                                                                                                                                              | Specify additional ClusterRoles to be granted to the accurate controller. "admin" is recommended to allow the controller to manage common namespace-scoped resources.                                                         |
| controller.config.annotationKeys                 | list   | `[]`                                                                                                                                                                              | Annotations to be propagated to sub-namespaces. It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.                                                                              |
| controller.config.labelKeys                      | list   | `[]`                                                                                                                                                                              | Labels to be propagated to sub-namespaces. It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.                                                                                   |
| controller.config.watches                        | list   | `[{"group":"rbac.authorization.k8s.io","kind":"Role","version":"v1"},{"group":"rbac.authorization.k8s.io","kind":"RoleBinding","version":"v1"},{"kind":"Secret","version":"v1"}]` | List of GVK for namespace-scoped resources that can be propagated. Any namespace-scoped resource is allowed.                                                                                                                  |
| controller.config.propagateAnnotationKeyExcludes | list   | `["*kubernetes.io/*"]`                                                                                                                                                            | Annotations to exclude when propagating resources. It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.                                                                           |
| controller.config.propagateLabelKeyExcludes      | list   | `["*kubernetes.io/*"]`                                                                                                                                                            | Labels to exclude when propagating resources. It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.                                                                                |
| controller.extraArgs                             | list   | `[]`                                                                                                                                                                              | Optional additional arguments.                                                                                                                                                                                                |
| controller.replicas                              | int    | `2`                                                                                                                                                                               | Specify the number of replicas of the controller Pod.                                                                                                                                                                         |
| controller.resources                             | object | `{"requests":{"cpu":"100m","memory":"20Mi"}}`                                                                                                                                     | Specify resources.                                                                                                                                                                                                            |
| controller.terminationGracePeriodSeconds         | int    | `10`                                                                                                                                                                              | Specify terminationGracePeriodSeconds.                                                                                                                                                                                        |
| webhook.allowCascadingDeletion                   | bool   | `false`                                                                                                                                                                           | Enable to allow cascading deletion of namespaces. Accurate webhooks will only allow deletion of a namespace with children if this option is enabled.                                                                          |
| image.pullPolicy                                 | string | `nil`                                                                                                                                                                             | Accurate image pullPolicy.                                                                                                                                                                                                    |
| image.repository                                 | string | `"ghcr.io/cybozu-go/accurate"`                                                                                                                                                    | Accurate image repository to use.                                                                                                                                                                                             |
| image.tag                                        | string | `{{ .Chart.AppVersion }}`                                                                                                                                                         | Accurate image tag to use.                                                                                                                                                                                                    |
| crds.enabled                                     | bool   | `true`                                                                                                                                                                            | Decides if the CRDs should be installed as part of the Helm installation.                                                                                                                                                     |
| crds.keep                                        | bool   | `true`                                                                                                                                                                            | Setting this to `true` will prevent Helm from uninstalling the CRD when the Helm release is uninstalled.                                                                                                                      |
| installCRDs                                      | bool   | `true`                                                                                                                                                                            | Controls if CRDs are automatically installed and managed as part of your Helm release. Deprecated: Use `crds.enabled` and `crds.keep` instead.                                                                                |

## Generate Manifests

You can use the `helm template` command to render manifests.

```bash
helm template --namespace accurate accurate accurate/accurate
```
