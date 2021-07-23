# Sub-namespace operations

Sub-namespaces is a feature of Accurate to allow tenant users to create Namespaces and delete the created Namespaces.

Sub-namespaces can be created in either a root Namespace or a sub-namespace.

In the following examples, `<name>` represents a Namespace name to be changed.
Likewise, `<parent>` represents a root or another sub-namespace.

## Setting a Namespace as a root Namespace

Using `kubectl accurate`:

```console
$ kubectl accurate ns set-type <name> root
```

Applying YAML manifests:

```console
apiVersion: v1
kind: Namespace
metadata:
  name: <name>
  labels:
    accurate.cybozu.com/type: root
```

## Reverting a root Namespace to a normal one

Using `kubectl accurate`:

```console
$ kubectl accurate ns set-type <name> none
```

Applying YAML manifests:

Remove `accurate.cybozu.com/type` label.

## Creating a sub-namespace

Using `kubectl accurate`:

```console
$ kubectl accurate sub create <name> <parent>
```

Applying YAML manifests:

```console
apiVersion: accurate.cybozu.com/v1
kind: SubNamespace
metadata:
  namespace: <parent>
  name: <name>
```

## Deleting a created sub-namespace

Using `kubectl accurate`:

```console
$ kubectl accurate sub delete <name>
```

Applying YAML manifests:

Delete the created SubNamespace object.

## Changing the parent of a sub-namespace

Only cluster admins can do this.

Using `kubectl accurate`:

```console
$ kubectl accurate sub move <name> <new-parent>
```

Applying YAML manifests:

```console
apiVersion: v1
kind: Namespace
metadata:
  name: <name>
  labels:
    accurate.cybozu.com/parent: <new-parent>
```

## Converting a normal Namespace to a sub-namespace

Only cluster admins can do this.

Using `kubectl accurate`:

```console
$ kubectl accurate sub graft <name> <parent>
```

Applying YAML manifests:

```console
apiVersion: v1
kind: Namespace
metadata:
  name: <name>
  labels:
    accurate.cybozu.com/parent: <parent>
```

## Converting a sub-namespace to a root Namespace

Only cluster admins can do this.

Using `kubectl accurate`:

```console
$ kubectl accurate sub cut <name>
```

Applying YAML manifests:

```console
apiVersion: v1
kind: Namespace
metadata:
  name: <name>
  labels:
    accurate.cybozu.com/type: root
    # and remove accurate.cybozu.com/parent label
```
