package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SubNamespaceStatus defines the observed state of SubNamespace
//+kubebuilder:validation:Enum=ok;conflict
type SubNamespaceStatus string

const (
	SubNamespaceOK       = SubNamespaceStatus("ok")
	SubNamespaceConflict = SubNamespaceStatus("conflict")
)

type SubNamespaceSpec struct {
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

//+kubebuilder:object:root=true

// SubNamespace is the Schema for the subnamespaces API
type SubNamespace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Spec is the spec of SubNamespace.
	// +optional
	Spec SubNamespaceSpec `json:"spec,omitempty"`

	// Status is the status of SubNamespace.
	// +optional
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
