# Annotations used by Innu

The table below is a list of annotations used by Innu.

| Key                                   | Value                    | Resource                       | Description                                                        |
| ------------------------------------- | ------------------------ | ------------------------------ | ------------------------------------------------------------------ |
| `innu.cybozu.com/from`                | Namespace name           | Copied or propagated resources | The namespace name from which the source resource was copied.      |
| `innu.cybozu.com/propagate`           | `"create"` or `"update"` | Namespace-scoped resources     | Specify propagation mode.                                          |
| `innu.cybozu.com/propagate-generated` | `"create"` or `"update"` | Namespace-scoped resources     | Specify propagation mode of generated resources.                   |
| `innu.cybozu.com/generated`           | `false`                  | Namespace-scoped resources     | The result of checking if this is generated from another resource. |
| `innu.cybozu.com/is-template`         | `"true"`                 | Namespace                      | Automatically added to template namespaces.                        |
