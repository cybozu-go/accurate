
### Custom Resources

* [SubNamespace](#subnamespace)

### Sub Resources

* [SubNamespaceList](#subnamespacelist)

#### SubNamespace

SubNamespace is the Schema for the subnamespaces API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| status | Status is the status of SubNamespace. | SubNamespaceStatus | false |

[Back to Custom Resources](#custom-resources)

#### SubNamespaceList

SubNamespaceList contains a list of SubNamespace

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][SubNamespace](#subnamespace) | true |

[Back to Custom Resources](#custom-resources)
