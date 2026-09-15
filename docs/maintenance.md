# Maintenance

## How to update supported Kubernetes

Accurate supports the three latest Kubernetes versions.
If a new Kubernetes is released, please update the following files.

- Update Kubernetes version in `versions.env`.
- Update kubectl version in `aqua.yaml`.
- Update `k8s.io/*` and `sigs.k8s.io/controller-runtime` packages version in `go.mod`.

If Kubernetes or controller-runtime API has changed, please fix the relevant source code.

## How to update dependencies

Renovate will create PRs that update dependencies when you [trigger the workflow with `workflow_dispatch`](https://github.com/cybozu-go/accurate/actions/workflows/renovate.yaml).
However, Kubernetes is only updated with patched versions.

Update GitHub Actions dependencies using pinact.

```sh
GITHUB_TOKEN="$(gh auth token)" pinact run --update --min-age 14
```
