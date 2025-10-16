# Setting up templates

Template is a feature of Accurate to propagate labels, annotations, and resources between normal Namespaces.

Any Namespace except for sub-namespaces can reference a template Namespace.
So, a template Namespace can reference another template Namespace.

In the following examples, `<name>` represents a Namespace name to be changed.
Likewise, `<template>` represents a template Namespace name.

## Setting a Namespace as a template

Using `kubectl accurate`:

```bash
kubectl accurate ns set-type <name> template
```

Applying YAML manifests:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: <name>
  labels:
    accurate.cybozu.com/type: template
```

## Reverting a template Namespace to a normal one

Using `kubectl accurate`:

```bash
kubectl accurate ns set-type <name> none
```

Applying YAML manifests:

Remove `accurate.cybozu.com/type` label.

## Setting a reference to a template Namespace

Using `kubectl accurate`:

```bash
kubectl accurate template set <name> <template>
```

Applying YAML manifests:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: <name>
  labels:
    accurate.cybozu.com/template: <template>
```

## Unsetting a reference to a template Namespace

Using `kubectl accurate`:

```bash
kubectl accurate template unset <name>
```

Applying YAML manifests:

Remove `accurate.cybozu.com/template` label.
