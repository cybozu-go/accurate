# Overview

Accurate is a Kubernetes controller to help operations in large soft multi-tenancy environments.

## Soft multi-tenancy in Kubernetes

Kubernetes does not provide multi-tenancy functions on its own.
It merely provides [Namespaces][Namespace] along with [Role-Based Access Control (RBAC)][RBAC] to isolate resources such as Pods.

_Soft multi-tenancy_ is a kind of technique to implement Namespace-based multi-tenancy on Kubernetes.
On the other hand, _hard multi-tenancy_ provides a virtual `kube-apiserver` for each tenant to isolate privileges completely.

In a soft multi-tenancy environment, a cluster admin grants privileges in a Namespace to a group of tenant users by creating RoleBinding object like this:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  namespace: tenant
  name: admin
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: admin
subjects:
- kind: Group
  name: group-for-tenant
  apiGroup: rbac.authorization.k8s.io
```

`admin` ClusterRole is [a built-in role](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#user-facing-roles) to give admin privileges on any kind of namespace-scope resources.
With this RoleBinding, users in `group-for-tenant` can freely create/edit/delete namespace-scope resources in `tenant` Namespace.

In many cases, a tenant needs to have multiple Namespaces to run multiple independent applications.
However, tenant users are not allowed to create or delete Namespaces because Namespace is a cluster-scope resource.
Otherwise, they would be able to delete other tenants' Namespaces!

## What is Accurate?

Accurate introduces a namespace-scope custom resource called **SubNamespace**.
With SubNamespace, tenant users can create a Namespace by creating a SubNamespace, and delete the created Namespace by deleting the SubNamespace.

The created Namespace is considered a child of the Namespace where the SubNamespace is created.
The child Namespace may inherit labels and annotations from its parent Namespace.

Accurate also propagates resources such as Role, RoleBinding, or Secret from a parent Namespace to its children Namespaces.
Without propagating Role/RoleBinding, the tenant user would be able to do nothing in newly created Namespaces.

## Features

- Resource propagation between namespaces

    Accurate can propagate any namespace-scope resource including custom resources between Namespaces.
    Moreover, Accurate can detect generated resources owned by another resource and propagate them.

- Inheriting labels and annotations creation/update from parent namespaces

    Namespace labels often play important roles.
    For example, [Pod Security Admission](https://github.com/kubernetes/website/blob/dev-1.22/content/en/docs/concepts/security/pod-security-admission.md#pod-security-admission-labels-for-namespaces), a new feature planned for Kubernetes 1.22, uses Namespace labels to control security policies.

- SubNamespace custom resource for tenant users

    This is the feature to allow tenant users to create and delete Namespaces by themselves.
    SubNamespaces can be created in either a root Namespace or a Namespace created by SubNamespace.
    A root Namespace is a Namespace labeled with `accurate.cybozu.com/type=root`.

- Template namespaces

    A template Namespace is a Namespace labeled with `accurate.cybozu.com/type=template`.
    Labels, annotations, and resources can be propagated from a template Namespace to other Namespaces referencing the template with `accurate.cybozu.com/template=<name>` label.

    This feature is implemented to allow resource propagation between normal Namespaces.

- `kubectl` plugin

    `kubectl-accurate` is a `kubectl` plugin to make the operations for Accurate easy.

[Namespace]: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
[RBAC]: https://kubernetes.io/docs/reference/access-authn-authz/rbac/
