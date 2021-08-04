# kubectl-accurate

`kubectl-accurate` is a `kubectl` plugin for Accurate.

## Features

- Hierarchical view of namespace trees
- Show the information about a namespace.
    - List of propagating/propagated resources in the namespace.
    - Root or not.
    - Template or not.
    - The parent namespace, if it is a sub-namespace.
    - The template namespace, if set.
- Operations for root namespaces
    - Make an independent namespace to a root namespace.
    - Make a root namespace back to an independent namespace, if it has no child sub-namespaces.
- Operations for setting a template namespace
- Operations for sub-namespaces
    - Create a sub-namespace under a root namespace or another sub-namespace.
    - Deleting a sub-namespace.
    - Move a sub-namespace under a different root or sub-namespace.
    - Convert an independent namespace to a sub-namespace.
    - Convert a sub-namespace to a root namespace.

## Generic options

`kubectl-accurate` takes the same generic options as `kubectl` including:

```
Flags:
      --as string                      Username to impersonate for the operation
      --as-group stringArray           Group to impersonate for the operation, this flag can be repeated to specify multiple groups.
      --cache-dir string               Default cache directory (default "$HOME/.kube/cache")
      --certificate-authority string   Path to a cert file for the certificate authority
      --client-certificate string      Path to a client certificate file for TLS
      --client-key string              Path to a client key file for TLS
      --cluster string                 The name of the kubeconfig cluster to use
      --context string                 The name of the kubeconfig context to use
      --insecure-skip-tls-verify       If true, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure
      --kubeconfig string              Path to the kubeconfig file to use for CLI requests.
  -n, --namespace string               If present, the namespace scope for this CLI request
      --request-timeout string         The length of time to wait before giving up on a single server request. Non-zero values should contain a corresponding time unit (e.g. 1s, 2m, 3h). A value of zero means don't timeout requests. (default "0")
  -s, --server string                  The address and port of the Kubernetes API server
      --tls-server-name string         Server name to use for server certificate validation. If it is not provided, the hostname used to contact the server is used
      --token string                   Bearer token for authentication to the API server
      --user string                    The name of the kubeconfig user to use
```

Note that `kubectl-accurate` does _not_ use the namespace given by `-n` / `--namespace` flag.
It always take namespace names as positional arguments.

## Commands

There is an alias for `namespace` sub-command that is `ns`.

### `list [ROOT]`

List namespace trees hierarchically.
If `ROOT` is given, only the tree starting from `ROOT` namespace is shown.

### `namespace describe NS`

Describe the information about a namespace `NS` related to Accurate.

### `namespace set-type NS TYPE`

Set the type of a namespace `NS` to `TYPE`.

Valid types are `root` or `template`.

To unset the type, specify `none` as the type.

### `template list [TEMPLATE]`

List template namespace trees hierarchically.
If TEMPLATE is not given, all root namespaces and their children will be shown.
If TEMPLATE is given, only the tree under the TEMPLATE namespace will be shown.

### `template set NS TEMPLATE`

Set `TEMPLATE` namespace as the template of `NS` namespace.

### `template unset NS`

Unset the template of `NS` namespace.

### `sub create NAME NS`

Create a [SubNamespace][] named `NAME` in `NS` namespace.

After that, Accurate will create a namespace `NAME` as a sub-namespace of `NS`.

### `sub delete NAME`

Delete a [SubNamespace][] named `NAME` in the parent namespace of `NAME` namespace.

After that, Accurate will delete `NAME` namespace.

### `sub move NS PARENT`

Move a sub-namespace `NS` to a different root or sub-namespace.

After that, Accurate will create [SubNamespace][] object in the new parent namespace.

### `sub graft NS PARENT`

Like `sub move`, but this converts a non-sub-namespace `NS` to a sub-namespace of `PARENT`.

`NS` must not be a root namespace or have a template.

### `sub cut NS`

Make a sub-namespace `NS` a new root namespace.
The child sub-namespaces under `NS` will be moved along with it.

Propagated resources with mode `update` in `NS` will be deleted.

[SubNamespace]: ./crd_subnamespace.md
