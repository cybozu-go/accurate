
### Custom Resources

* [SubNamespace](#subnamespace)

### Sub Resources

* [SubNamespaceList](#subnamespacelist)
* [SubNamespaceSpec](#subnamespacespec)

#### SubNamespace

SubNamespace is the Schema for the subnamespaces API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec | Spec is the spec of SubNamespace. | [SubNamespaceSpec](#subnamespacespec) | false |
| status | Status is the status of SubNamespace. | SubNamespaceStatus | false |

[Back to Custom Resources](#custom-resources)

#### SubNamespaceList

SubNamespaceList contains a list of SubNamespace

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][SubNamespace](#subnamespace) | true |

[Back to Custom Resources](#custom-resources)

#### SubNamespaceSpec



| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| labels |  | map[string]string | false |
| annotations |  | map[string]string | false |

[Back to Custom Resources](#custom-resources)
