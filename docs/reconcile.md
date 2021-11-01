# How Accurate reconciles resources

Accurate primarily watches Namespaces and SubNamespaces.
In addition, it needs to watch any kind of resources specified in its config file.

## SubNamespace (custom resource)

- For a new SubNamespace, Accurate creates a sub-namespace.
- For a deleting SubNamespace, Accurate deletes the sub-namespace if the sub-namespace exists and its `accurate.cybozu.com/parent` is the same as `metadata.namespace` of SubNamespace.

## Namespaces

### Namespaces that are labeled with `accurate.cybozu.com/template`

These namespaces reference a template namespace and propagate the labels, annotations, and watched resources from the template namespace.

- Accurate should propagate labels and/or annotations from the template namespace.
- Accurate should create copies of resources in the template namespace whose `accurate.cybozu.com/propagate` annotation is `create` if they are missing.
- Accurate should create or update copies of resources in the template namespace whose `accurate.cybozu.com/propagate` annotation is `update` if they are missing or different.
- Accurate should delete resources in the reconciling namespace that are annotated with `accurate.cybozu.com/propagate=update` provided that:
    - the value of `accurate.cybozu.com/from` annotation is not the template namespace name, or
    - there is not a resource of the same kind and the same name in the template namespace.

### Namespaces w/o `accurate.cybozu.com/type` and `accurate.cybozu.com/template` labels

If these labels are removed from the Namespace, Accurate should delete propagated resources with mode == `update`.

### Template namespace

Template namespaces are namespaces labeled with `accurate.cybozu.com/type=template`.

- Accurate should propagate labels and/or annotations to namespaces that references the template namespace with `accurate.cybozu.com/template` label.

### Root namespace

Root namespaces are namespaces labeled with `accurate.cybozu.com/type=root`.

- Accurate should propagate labels and/or annotations to its sub-namespaces.

### Sub-namespace

Sub-namespaces are namespaces created by Accurate.
Sub-namespaces have `accurate.cybozu.com/parent` label.

- Accurate should propagate labels and/or annotations from the parent namespace.
- Accurate should propagate labels and/or annotations of the reconciling namespace to its sub-namespaces, if any.
- Accurate should create copies of resources in the parent namespace whose `accurate.cybozu.com/propagate` annotation is `create` if they are missing.
- Accurate should create or update copies of resources in the parent namespace whose `accurate.cybozu.com/propagate` annotation is `update` if they are missing or different.
- Accurate should delete resources in the reconciling namespace that are annotated with `accurate.cybozu.com/propagate=update` provided that:
    - the value of `accurate.cybozu.com/from` annotation is not the parent namespace name, or
    - there is not a resource of the same kind and the same name in the parent namespace.

## Watched namespace-scoped resources

Any namespace-scoped resource can be propagated from a template or from a parent namespace.

### Resources annotated with `accurate.cybozu.com/from`

These resources are propagated from a parent or a template namespace.
The annotation value is the parent namespace name.

- If the parent resource exists and is annotated with `accurate.cybozu.com/propagate=update`, Accurate compares the resource with the parent resource, and if they differ, updates the resource.
- If the resource is annotated with `accurate.cybozu.com/propagate=update` and there isn't a resource of the same kind and the same name in the parent namespace, Accurate deletes the resource.

The last rule is for cases where the parent resource is deleted while the controller is stopped.
With this rule, Accurate can delete such orphaned resources when the controller starts.

### Resources annotated with `accurate.cybozu.com/propagate`

These resources can be propagated to other namespaces.

- If the resource exists and the annotation value is `create`, Accurate creates a copy in all sub-namespaces if missing.
- If the resource exists and the annotation value is `update`, Accurate creates or updates a copy in all sub-namespaces if missing or different.
- When a resource is deleted, Accurate checks sub-namespaces and delete the resource of the same kind and the same name if the resource is annotated with `accurate.cybozu.com/propagate=update`.

### Resources owned by another resource that is annotated with `accurate.cybozu.com/propagate-generated`

Accurate annotates the resource with `accurate.cybozu.com/propagate`.
The annotation value is the same as `accurate.cybozu.com/propagate-generated` annotation.
