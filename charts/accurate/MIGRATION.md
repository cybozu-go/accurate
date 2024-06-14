# Migrate from kustomize to Helm

This document describes the steps to migrate from kustomize to Helm.

## Install Helm chart

There is no significant difference between the manifests installed by kustomize and those installed by Helm.

If a resource with the same name already exists in the Cluster, Helm will not be able to create the resource.

```console
$ helm repo add accurate https://cybozu-go.github.io/accurate/
$ helm repo update
$ helm install --namespace accurate accurate accurate/accurate
Error: rendered manifests contain a resource that already exists. Unable to continue with install: ServiceAccount "accurate-controller-manager" in namespace "accurate" exists and cannot be imported into the current release: invalid ownership metadata; label validation error: missing key "app.kubernetes.io/managed-by": must be set to "Helm"; annotation validation error: missing key "meta.helm.sh/release-name": must be set to "accurate"; annotation validation error: missing key "meta.helm.sh/release-namespace": must be set to "accurate"
```

Before installing Helm chart, you need to manually delete the resources.
You do not need to delete Namespace, CRD and SubNamespace custom resources at this time.

```console
$ helm template --namespace accurate accurate accurate/accurate | kubectl delete -f -
serviceaccount "accurate-controller-manager" deleted
clusterrole.rbac.authorization.k8s.io "accurate-manager-role" deleted
clusterrole.rbac.authorization.k8s.io "accurate-subnamespace-editor-role" deleted
clusterrole.rbac.authorization.k8s.io "accurate-subnamespace-viewer-role" deleted
clusterrolebinding.rbac.authorization.k8s.io "accurate-manager-admin" deleted
clusterrolebinding.rbac.authorization.k8s.io "accurate-manager-rolebinding" deleted
role.rbac.authorization.k8s.io "accurate-leader-election-role" deleted
rolebinding.rbac.authorization.k8s.io "accurate-leader-election-rolebinding" deleted
service "accurate-webhook-service" deleted
deployment.apps "accurate-controller-manager" deleted
certificate.cert-manager.io "accurate-serving-cert" deleted
issuer.cert-manager.io "accurate-selfsigned-issuer" deleted
mutatingwebhookconfiguration.admissionregistration.k8s.io "accurate-mutating-webhook-configuration" deleted
validatingwebhookconfiguration.admissionregistration.k8s.io "accurate-validating-webhook-configuration" deleted
Error from server (NotFound): error when deleting "STDIN": configmaps "accurate-config" not found # This is because the ConfigMap created by ConfigMapGeneraor will be suffixed. There is no problem to ignore it.
```

Then install Helm chart again.

```console
$ helm install --namespace accurate accurate accurate/accurate
NAME: accurate
LAST DEPLOYED: Fri Aug 20 10:12:03 2021
NAMESPACE: accurate
STATUS: deployed
REVISION: 1
TEST SUITE: None
```

## Configuration

Helm uses the values file to configure Accurate config file.

```yaml
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

    # controller.config.subNamespaceLabelKeys -- Labels to be propagated to sub-namespaces from SubNamespace resource.
    # It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
    # https://pkg.go.dev/path#Match
    subNamespaceLabelKeys: []
    # - app

    # controller.config.subNamespaceAnnotationKeys -- Annotations to be propagated to sub-namespaces from SubNamespace resource.
    # It is also possible to specify a glob pattern that can be interpreted by Go's "path.Match" func.
    # https://pkg.go.dev/path#Match
    subNamespaceAnnotationKeys: []
    # - foo.bar/baz

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
```

Optional: If you have customized RBAC, you can use `additionalRBAC`.

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

The values file can be specified with the `-f` option when you install Helm chart.

```console
$ helm install --create-namespace --namespace accurate accurate accurate/accurate -f values.yaml
```

There are several other configurable items besides the Accurate config file. See [README.md](./README.md) for details.
