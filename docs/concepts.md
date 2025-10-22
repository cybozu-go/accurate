# Concepts

## Namespace types

Accurate defines the following types of Namespaces:

- Template: Namespace labeled with `accurate.cybozu.com/type=template`
- Root: Namespace labeled with `accurate.cybozu.com/type=root`
- Sub-namespace: Namespace labeled with `accurate.cybozu.com/parent=<name>`

Any Namespace other than sub-namespaces can reference a template Namespace with `accurate.cybozu.com/template=<name>` label.

Sub-namespace can reference a root or another sub-namespace as its parent.

When configured to do so, Accurate propagates the Namespace labels, annotations, and namespace-scope resources from a referenced Namespace to referencing Namespaces.

Circular references are prohibited by an admission webhook.

## Resource propagation

Accurate propagates any namespace-scope resource that are annotated with `accurate.cybozu.com/propagate=<mode>`.

Mode is one of the following:

- `create`: the resource will be created in referencing Namespaces if missing.
- `update`: the resource will be created in referencing Namespaces if missing, or will be updated if not identical, or will be deleted when the resource in the referenced Namespace is deleted.

## Propagating generated resources (DEPRECATED)

<div class="warning">
Propagating generated resources is a deprecated feature and is subject for
removal soon.
</div>

If a resource annotated with `accurate.cybozu.com/propagate-generated=<mode>` creates a resource and set an owner reference in the created resource, Accurate automatically adds `accurate.cybozu.com/propagate=<mode>` to the created resource.

This can be used, for example, to propagate Secret created from [SealedSecret][].

[SealedSecret]: https://github.com/bitnami-labs/sealed-secrets
