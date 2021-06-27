# How Innu reconciles resources

Innu primarily watches Namespaces and SubNamespaces.
In addition, it needs to watch any kind of resources specified in its config file.

## SubNamespace (custom resource)

- For a new SubNamespace, Innu creates a sub-namespace.
- For a deleting SubNamespace, Innu deletes the sub-namespace if the sub-namespace exists and its `innu.cybozu.com/parent` is the same as `metadata.namespace` of SubNamespace.

## Namespaces

### Namespaces that are not sub-namespaces

These namespaces can set a template namespace.
Innu reconciles resources as follows only when the namespace is labeled with `innu.cybozu.com/template`.

- Innu copies labels and annotations of the template namespace to the reconciling namespace, and
- annotates the template namespace with `innu.cybozu.com/is-template=true`, and
- if there are resources annotated with `innu.cybozu.com/propagate=create` in the template namespace, creates copies of them in the reconciling namespace when they are missing, and
- if there are resources annotated with `innu.cybozu.com/propagate=update` in the template namespace, creates or updates copies of them in the reconciling namespace when they are missing or different from those in the template namespace, and
- delete resources in the reconciling namespace that are annotated with `innu.cybozu.com/propagate=update` provided that:
    - the value of `innu.cybozu.com/from` annotation is not the template namespace name, or
    - there is not a resource of the same kind and the same name in the template namespace.

### Template namespace

Template namespaces are namespaces annotated with `innu.cybozu.com/is-template=true` (see above).

- Innu should propagate labels and/or annotations to namespaces that references the template namespace with `innu.cybozu.com/template` label.

### Root namespace

Root namespaces are namespaces labeled with `innu.cybozu.com/root=true`.

- Innu should propagate labels and/or annotations to its sub-namespaces.

### Sub-namespace

Sub-namespaces are namespaces created by Innu.
Sub-namespaces have `innu.cybozu.com/parent` label.

- Innu should propagate labels and/or annotations from the parent namespace.
- Innu should propagate labels and/or annotations of the reconciling namespace to its sub-namespaces, if any.
- Innu should create copies of resources in the parent namespace whose `innu.cybozu.com/propagate` annotation is `create` if they are missing.
- Innu should create or update copies of resources in the parent namespace whose `innu.cybozu.com/propagate` annotation is `update` if they are missing or different.
- Innu should delete resources annotated with `innu.cybozu.com/propagate=update` if the origin resource is missing in the parent namespace.

## Watched namespace-scoped resources

Any namespace-scoped resource can be propagated from a template or from a parent namespace.

### Resources annotated with `innu.cybozu.com/from`

These resources are propagated from a parent or a template namespace.
The annotation value is the parent namespace name.

- If the parent resource exists and is annotated with `innu.cybozu.com/propagate=update`, Innu compares the resource with the parent resource, and if they differ, updates the resource.
- If the resource is annotated with `innu.cybozu.com/propagate=update` and there isn't a resource of the same kind and the same name in the parent namespace, Innu deletes the resource.

The last rule is for cases where the parent resource is deleted while the controller is stopped.
With this rule, Innu can delete such orphaned resources when the controller starts.

### Resources annotated with `innu.cybozu.com/propagate`

These resources can be propagated to other namespaces.

- If the resource exists and the annotation value is `create`, Innu creates a copy in all sub-namespaces if missing.
- If the resource exists and the annotation value is `update`, Innu creates or updates a copy in all sub-namespaces if missing or different.
- When a resource is deleted, Innu checks sub-namespaces and delete the resource of the same kind and the same name if the resource is annotated with `innu.cybozu.com/propagate=update`.

### Resources owned by another resource that is annotated with `innu.cybozu.com/propagate-generated`

Innu annotates the resource with `innu.cybozu.com/propagate`.
The annotation value is the same as `innu.cybozu.com/propagate-generated` annotation.
