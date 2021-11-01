# Accurate Helm Chart

## How to use Accurate Helm repository

You need to add this repository to your Helm repositories:

```console
helm repo add accurate https://cybozu-go.github.io/accurate/
helm repo update
```

## Quick start

### Installing cert-manager

```console
$ curl -fsL https://github.com/jetstack/cert-manager/releases/latest/download/cert-manager.yaml | kubectl apply -f -
```

### Installing the Chart

> NOTE:
>
> This installation method requires cert-manager to be installed beforehand.

To install the chart with the release name `accurate` using a dedicated namespace(recommended):

```console
$ helm install --create-namespace --namespace accurate accurate accurate/accurate
```

Specify parameters using `--set key=value[,key=value]` argument to `helm install`.

Alternatively a YAML file that specifies the values for the parameters can be provided like this:

```console
$ helm install --create-namespace --namespace accurate accurate -f values.yaml accurate/accurate
```

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| controller.additionalRBAC.rules | list | `[]` | Specify the RBAC rules to be added to the controller. ClusterRole and ClusterRoleBinding are created with the names `{{ release name }}-additional-resources`. The rules defined here will be used for the ClusterRole rules. |
| controller.config.annotationKeys | list | `[]` | Annotations to be propagated to sub-namespaces. It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func. |
| controller.config.labelKeys | list | `[]` | Labels to be propagated to sub-namespaces. It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func. |
| controller.config.watches | list | `[{"group":"rbac.authorization.k8s.io","kind":"Role","version":"v1"},{"group":"rbac.authorization.k8s.io","kind":"RoleBinding","version":"v1"},{"kind":"Secret","version":"v1"}]` | List of GVK for namespace-scoped resources that can be propagated. Any namespace-scoped resource is allowed. |
| controller.extraArgs | list | `[]` | Optional additional arguments. |
| controller.replicas | int | `2` | Specify the number of replicas of the controller Pod. |
| controller.resources | object | `{"requests":{"cpu":"100m","memory":"20Mi"}}` | Specify resources. |
| controller.terminationGracePeriodSeconds | int | `10` | Specify terminationGracePeriodSeconds. |
| image.pullPolicy | string | `nil` | Accurate image pullPolicy. |
| image.repository | string | `"ghcr.io/cybozu-go/accurate"` | Accurate image repository to use. |
| image.tag | string | `{{ .Chart.AppVersion }}` | Accurate image tag to use. |

## Generate Manifests

You can use the `helm template` command to render manifests.

```console
$ helm template --namespace accurate accurate accurate/accurate
```

## Upgrade CRDs

There is no support at this time for upgrading or deleting CRDs using Helm.
Users must manually upgrade the CRD if there is a change in the CRD used by Accurate.

https://helm.sh/docs/chart_best_practices/custom_resource_definitions/#install-a-crd-declaration-before-using-the-resource

## Release Chart

Accurate Helm Chart will be released independently.
This will prevent the Accurate version from going up just by modifying the Helm Chart.

You must change the version of `Chart.yaml` when making changes to the Helm Chart.

Pushing a tag like `chart-v<chart version>` will cause GitHub Actions to release chart.
Chart versions are expected to follow [Semantic Versioning](https://semver.org/).
If the chart version in the tag does not match the version listed in `Chart.yaml`, the release will fail.
