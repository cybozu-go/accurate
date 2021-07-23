[![GitHub release](https://img.shields.io/github/release/cybozu-go/accurate.svg?maxAge=60)][releases]
[![CI](https://github.com/cybozu-go/accurate/actions/workflows/ci.yaml/badge.svg)](https://github.com/cybozu-go/accurate/actions/workflows/ci.yaml)
[![PkgGoDev](https://pkg.go.dev/badge/github.com/cybozu-go/accurate?tab=overview)](https://pkg.go.dev/github.com/cybozu-go/accurate?tab=overview)
[![Go Report Card](https://goreportcard.com/badge/github.com/cybozu-go/accurate)](https://goreportcard.com/report/github.com/cybozu-go/accurate)

# Accurate

Accurate is a Kubernetes controller for multi-tenancy.

Accurate resembles [Hierarchical Namespace Controller (HNC)][HNC].
It propagates resources between namespaces and allows tenant users to create/delete sub-namespaces.

**Project Status**: Beta

## Features

- Resource propagation between namespaces
    - Any namespace-scoped resource can be propagated.
    - Generated resources can be automatically checked and propagated.
- Inheriting labels and annotations from parent namespaces
- Template namespaces
- SubNamespace custom resource for tenant users
- `kubectl` plugin

## Comparison to Hierarchical Namespace Controller (HNC)

Both Accurate and HNC aim the same goal -- to provide better namespace usability on soft multi-tenancy Kubernetes environments.

Accurate is more accurate than HNC in propagating resources because Accurate adopts an opt-in manner while HNC adopts an opt-out manner.
With Accurate, only resources annotated with `accurate.cybozu.com/propagate` will be propagated.
With HNC, all resources will be propagated except for ones that are specially annotated.

Suppose you want to propagate only [a Secret for pulling private images](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).
With HNC, this can be quite difficult because Secrets are often generated from another resource.
Such generated Secrets are often not possible to have custom annotations.
As a consequence, such Secrets would be propagated to sub-namespaces, which may cause security problems.

There are many other differences between Accurate and HNC.
Please check them in [the documentation][doc].

## Demo

Run and try Accurate on a [kind (Kubernetes-In-Docker)][kind] cluster as follows:

1. Prepare a Linux box running Docker.
2. Checkout this repository.

    ```console
    $ git clone https://github.com/cybozu-go/accurate
    ```

3. Go to `e2e` directory, setup shell variables, and execute `make start`.

    ```console
    $ cd e2e
    $ PATH=$(cd ..; pwd)/bin:$PATH
    $ KUBECONFIG=$(pwd)/.kubeconfig
    $ export KUBECONFIG
    $ make start
    ```

4. Create a root namespace and a sub-namespace using `kubectl accurate`.

    ```console
    $ kubectl create ns root1
    $ kubectl accurate ns set-type root1 root
    $ kubectl accurate sub create sub1 root1
    $ kubectl accurate list
    $ kubectl accurate ns describe root1
    $ kubectl accurate ns describe sub1
    ```

5. Create a Secret in `root1` and see it will be propagated to `sub1`.

    ```console
    $ kubectl -n root1 create secret generic s1 --from-literal=foo=bar
    $ kubectl -n root1 annotate secrets s1 accurate.cybozu.com/propagate=update
    $ sleep 1
    $ kubectl -n sub1 get secrets
    ```

6. Stop the kind cluster.

    ```console
    $ make stop
    ```

## Documentation

Read the documentation at https://cybozu-go.github.io/accurate .

[releases]: https://github.com/cybozu-go/accurate/releases
[HNC]: https://github.com/kubernetes-sigs/hierarchical-namespaces
[doc]: https://cybozu-go.github.io/accurate
[kind]: https://kind.sigs.k8s.io/
