
### Custom Resources

* [SubNamespace](#subnamespace)

### Sub Resources

* [SubNamespaceList](#subnamespacelist)
* [SubNamespaceMoveSpec](#subnamespacemovespec)
* [SubNamespaceSpec](#subnamespacespec)
* [SubNamespaceStatus](#subnamespacestatus)

#### SubNamespace

SubNamespace is the Schema for the subnamespaces API

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ObjectMeta | false |
| spec | Spec is the spec of SubNamespace. | [SubNamespaceSpec](#subnamespacespec) | false |
| status | Status is the status of SubNamespace. | [SubNamespaceStatus](#subnamespacestatus) | false |

[Back to Custom Resources](#custom-resources)

#### SubNamespaceList

SubNamespaceList contains a list of SubNamespace

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| metadata |  | metav1.ListMeta | false |
| items |  | [][SubNamespace](#subnamespace) | true |

[Back to Custom Resources](#custom-resources)

#### SubNamespaceMoveSpec

SubNamespaceMoveSpec defines a move between parent namespaces.

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| sourceParent | SourceParent is the current parent namespace of the sub-namespace. | string | false |
| targetParent | TargetParent is the desired parent namespace of the sub-namespace. | string | false |

[Back to Custom Resources](#custom-resources)

#### SubNamespaceSpec

SubNamespaceSpec defines the desired state of SubNamespace

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| labels | Labels are the labels to be propagated to the sub-namespace | map[string]string | false |
| annotations | Annotations are the annotations to be propagated to the sub-namespace. | map[string]string | false |
| move | Move specifies a requested or accepted move of this SubNamespace. | *[SubNamespaceMoveSpec](#subnamespacemovespec) | false |

[Back to Custom Resources](#custom-resources)

#### SubNamespaceStatus

SubNamespaceStatus defines the observed state of SubNamespace

| Field | Description | Scheme | Required |
| ----- | ----------- | ------ | -------- |
| observedGeneration | The generation observed by the object controller. | int64 | false |
| conditions | Conditions represent the latest available observations of an object's state | []metav1.Condition | false |

[Back to Custom Resources](#custom-resources)
