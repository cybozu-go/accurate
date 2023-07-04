# Maintenance

## How to update supported Kubernetes

Accurate supports the three latest Kubernetes versions.
If a new Kubernetes is released, please update the following files.

- Update Kubernetes version in `e2e/Makefile` and `.github/workflows/ci.yaml`.
- Update kubectl version in `aqua.yaml`.
- Update `k8s.io/*` and `sigs.k8s.io/controller-runtime` packages version in `go.mod`.

If Kubernetes or controller-runtime API has changed, please fix the relevant source code.

## How to update dependencies

Renovate will create PRs that update dependencies once a week.
However, Kubernetes is only updated with patched versions.
