# Design notes

Innu aims to implement functionalities found in [Hierarchical Namespace Controller (HNC)][HNC], but in a different way.

Innu stands for "**IN**heritable **N**amespaces for multi-tenant **U**sers".

## Why do we need another HNC in the first place?

Some of the HNC designs and specifications contradict our use cases.

- HNC opt-outs resources when propagating them.
    - For safety and preciseness, we need opt-in propagation.
- HNC opt-outs root namespaces.
    - For easier-maintenance, we need opt-in root namespaces.
- HNC does not propagate namespace labels and annotations.
    - We need to propagate some namespace labels/annotations.

Since these are fundamentally different requirements, we decided to develop our own solution.

## Goals

- Any namespace-scoped resource can be copied or propagated
    - The kinds of resources are given by the configuration file of Innu.
    - Only resources annotated with `innu.cybozu.com/propagate: <mode>` will be propagated.
    - Of course, Innu controller needs to be privileged to manage them.
- Support the following propagation modes:
    - `create`: if the resource does not exist, copy the resource from the parent namespace.
    - `update`: if the resource is missing or different from the parent namespace, create or update it.  If the parent resource is deleted, the copy will also be deleted.
- Opt-in root namespaces
    - Namespaces labeled with `innu.cybozu.com/root: "true"` will become roots.
- Tenant users can create and delete sub-namespaces.
    - By creating and deleting a custom resource in a root namespace or one of its sub-namespaces.
    - If a sub-namespace has one or more sub-namespaces, Innu prevents the deletion of the sub-namespace.
- Template namespace
    - Namespaces that are not sub-namespaces can specify a template namespace by `innu.cybozu.com/template: <name>` label.
    - Resources in the template namespace will be propagated to the namespace according to the mode.
- Propagate generated resources
    - Resources created and controlled by another resource can be automatically propagated.
    - The generator resource should be annotated with `innu.cybozu.com/propagate-generated: <mode>`.
- Propagate labels and annotations of parent or template namespaces
    - The label/annotation keys are given through the configuration file of Innu.
    - Only labels/annotations having a matching key will be propagated.
- Admins can change the parent namespace of a sub-namespace.

Non-goals:

- Hierarchy between full namespaces
    - A full namespace is a namespace that is not created by Innu.
    - Innu don't provide a way to propagate resources between full namespaces.
    - Template namespace provides nearly the same functionality while eliminating complexities.
- `kubectl` plugin
    - Desirable, but not necessary.

## Things to be avoided

Innu should make sure not to cause the following things:

- Circular references among sub-namespaces.
- Deleting `innu.cybozu.com/root` label from root namespaces having one or more sub-namespaces.
- Dangling sub-namespaces (sub-namespaces whose parent is missing).
- Creating a sub-namespace under a non-root and non-sub- namespace.
- Changing a sub-namespace that has child sub-namespaces to a non-root namespace.

These can be prevented with validating admission webhooks for SubNamespace and Namespace.

### No webhooks for propagated resources

Innu does not use admission webhooks for resources propagated from a parent namespace to its sub-namespaces.
The decision was made from the following points:

- Since Innu can propagate any namespace-scoped resource, adding an admission webhook for them might cause troubles.

    For instance, admission webhooks for Pods may cause chicken-and-egg problem upon bootstrapping.

- Avoid interrupting users who do not expect limitations from Innu.

## SubNamespaces are only informational in propagating resources

Innu does not rely on SubNamespace resources to look up sub-namespaces of a namespace or to find the parent of a sub-namespace.

By doing so, Innu can easily migrate sub-namespaces under another root/sub-namespace.

[HNC]: https://github.com/kubernetes-sigs/hierarchical-namespaces
