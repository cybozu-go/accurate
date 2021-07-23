# Labels used by Accurate

The table below is a list of labels used by Accurate.

| Key                            | Value                | Resource                                       | Description                  |
| ------------------------------ | -------------------- | ---------------------------------------------- | ---------------------------- |
| `accurate.cybozu.com/type`     | `template` or `root` | Namespace                                      | The type of namespace.       |
| `accurate.cybozu.com/template` | Namespace name       | Namespace                                      | The template namespace name. |
| `accurate.cybozu.com/parent`   | Namespace name       | Namespace                                      | The parent namespace name.   |
| `app.kubernetes.io/created-by` | `accurate`           | Copied or propagated resources, sub-namespaces | Informational                |
