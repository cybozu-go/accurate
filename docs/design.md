# Design notes

- [Overview](#overview)
- [Why do we need another namespace controller in the first place?](#why-do-we-need-another-namespace-controller-in-the-first-place)
- [Goals](#goals)
- [Things to be avoided](#things-to-be-avoided)
  - [No webhooks for propagated resources](#no-webhooks-for-propagated-resources)
- [SubNamespaces are not related to parent-child relationships](#subnamespaces-are-not-related-to-parent-child-relationships)

## Overview

Accurate aims to implement functionalities found in [Hierarchical Namespace Controller (HNC)][HNC], but in a different way.

Accurate provides 1) resource-propagation between namespaces, and 2) sub-namespace concept for multi-tenancy.
Since we consider resource-propagation alone is highly useful, the feature is available between any namespaces.

## Why do we need another namespace controller in the first place?

Some of the HNC designs and specifications contradict our use cases.

- HNC opt-outs resources when propagating them.
    - For safety and accuracy, we need opt-in propagation.
- HNC opt-outs root namespaces.
    - For easier-maintenance, we need opt-in root namespaces.
- HNC does not propagate namespace labels and annotations.
    - We need to propagate some namespace labels/annotations.

Since these are fundamentally different requirements, we decided to develop our own solution.

## Goals

- Any namespace-scoped resource can be copied or propagated
    - The kinds of resources are given by the configuration file of Accurate.
    - Only resources annotated with `accurate.cybozu.com/propagate: <mode>` will be propagated.
    - Of course, Accurate controller needs to be privileged to manage them.
- Support the following propagation modes:
    - `create`: if the resource does not exist, copy the resource from the parent namespace.
    - `update`: if the resource is missing or different from the parent namespace, create or update it.  If the parent resource is deleted, the copy will also be deleted.
- ⚠️ Propagate generated resources (DEPRECATED)
    - Resources created and controlled by another resource can be automatically propagated.
    - The generator resource should be annotated with `accurate.cybozu.com/propagate-generated: <mode>`.
- Propagate labels and annotations of parent or template namespaces
    - The label/annotation keys are given through the configuration file of Accurate.
    - Only labels/annotations specified in the configuration file of Accurate will be propagated.
    - Label/annotation deletions from parent or template namespaces will not be propagated.
- Opt-in root namespaces
    - Only namespaces labeled with `accurate.cybozu.com/type: root` can be the root of a namespace tree.
- Tenant users can create and delete sub-namespaces by creating and deleting a custom resource in a root or a sub-namespace.
    - If a namespace has one or more sub-namespaces, Accurate prevents the deletion of the namespace - unless allow cascading deletion of namespaces is enabled.
- Template namespace
    - Namespaces that are not a sub-namespace can specify a template from which labels, annotations, and resources can be propagated.
- Admins can change the parent namespace of a sub-namespace.

## Things to be avoided

Accurate prevents the following problems by a validating admission webhook for Namespace.

- Circular references among namespaces.
- Allowing a sub-namespace to set a template (sub-namespaces should inherit things only from the parent).
- Marking a sub-namespace as a root namespace.
- Deleting `accurate.cybozu.com/type=root` label from root namespaces having one or more sub-namespaces.
- Deleting `accurate.cybozu.com/type=template` label from template namespaces having one or more instance namespaces.
- Dangling sub-namespaces (sub-namespaces whose parent namespace is missing).
- Dangling instance namespaces (namespaces whose template namespace is missing).
- Changing a sub-namespace to a non-root namespace when it has child sub-namespaces.

Accurate prevents the following problem by a validating admission webhook for SubNamespace.

- Creating a SubNamespace object in a non-root and non-sub- namespace.

### No webhooks for propagated resources

Accurate does not use admission webhooks for resources propagated from a parent namespace to its sub-namespaces.
The decision was made from the following points:

- Since Accurate can propagate any namespace-scoped resource, adding an admission webhook for them might cause troubles.

    For instance, admission webhooks for Pods may cause chicken-and-egg problem upon bootstrapping.

- Avoid interrupting users who do not expect limitations from Accurate.

## SubNamespaces are not related to parent-child relationships

Accurate does not rely on SubNamespace resources to look up sub-namespaces of a namespace or to find the parent of a sub-namespace.

By doing so, Accurate can easily restructure existing namespaces.

[HNC]: https://github.com/kubernetes-sigs/hierarchical-namespaces
