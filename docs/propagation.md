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

## Annotating a resource to propagate resources created from it (DEPRECATED)

<div class="warning">
Propagating generated resources is a deprecated feature and is subject for
removal soon. Commonly used tools like cert-manager and sealed-secrets now
provide features for adding annotations/labels to resources created from
user-facing custom resources. These features can be used for migration to
ensure the standard `accurate.cybozu.com/propagate` annotation is added to
generated resources.
</div>

For example, a Secret created from cert-manager's Certificate can automatically be propagated.

To do this, Certificate should be annotated with `accurate.cybozu.com/propagate-generated=<mode>` at the time of creation.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  namespace: default
  name: example-cert
  annotations:
    accurate.cybozu.com/propagate-generated: <mode>
spec:
  ...
```

`accurate-controller` needs to be able to get Certificate objects.
