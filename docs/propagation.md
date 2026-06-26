# Propagating resources

Accurate propagates only resources annotated with `accurate.cybozu.com/propagate=<mode>`.

The Group/Version/Kind of the resource must be listed in the [configuration file](config.md).

In the following examples, `<mode>` represents either `create` or `update`.
Read [Concepts](concepts.md) about the propagation modes.

## Annotating a resource for propagation

The following is an example to propagate Secrets.

Using `kubectl`:

```bash
kubectl annotate secrets <name> accurate.cybozu.com/propagate=<mode>
```

Applying YAML manifests:

```console
apiVersion: v1
kind: Secret
metadata:
  namespace: default
  name: <name>
  annotations:
    accurate.cybozu.com/propagate: <mode>
```

