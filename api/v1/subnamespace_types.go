package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SubNamespaceSpec defines the desired state of SubNamespace
type SubNamespaceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of SubNamespace. Edit subnamespace_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// SubNamespaceStatus defines the observed state of SubNamespace
type SubNamespaceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SubNamespace is the Schema for the subnamespaces API
type SubNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubNamespaceSpec   `json:"spec,omitempty"`
	Status SubNamespaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SubNamespaceList contains a list of SubNamespace
type SubNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SubNamespace `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SubNamespace{}, &SubNamespaceList{})
}
