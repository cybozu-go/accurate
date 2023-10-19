# Annotations used by Accurate

The table below is a list of annotations used by Accurate.

| Key                                       | Value                    | Resource                       | Description                                                        |
| ----------------------------------------- | ------------------------ | ------------------------------ | ------------------------------------------------------------------ |
| `accurate.cybozu.com/from`                | Namespace name           | Copied or propagated resources | The namespace name from which the source resource was copied.      |
| `accurate.cybozu.com/propagate`           | `"create"` or `"update"` | Namespace-scoped resources     | Specify propagation mode.                                          |
| `accurate.cybozu.com/propagate-generated` ⚠️ | `"create"` or `"update"` | Namespace-scoped resources     | `DEPRECATED` Specify propagation mode of generated resources.                   |
| `accurate.cybozu.com/generated` ⚠️          | `false`                  | Namespace-scoped resources     | `DEPRECATED` The result of checking if this is generated from another resource. |
