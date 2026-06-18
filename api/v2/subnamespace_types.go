package v2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SubNamespaceStatus defines the observed state of SubNamespace
type SubNamespaceStatus struct {
	// The generation observed by the object controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions represent the latest available observations of an object's state
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// SubNamespaceSpec defines the desired state of SubNamespace
type SubNamespaceSpec struct {
	// Labels are the labels to be propagated to the sub-namespace
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations are the annotations to be propagated to the sub-namespace.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Move specifies a requested or accepted move of this SubNamespace.
	// +optional
	Move *SubNamespaceMoveSpec `json:"move,omitempty"`
}

// SubNamespaceMoveSpec defines a move between parent namespaces.
type SubNamespaceMoveSpec struct {
	// SourceParent is the current parent namespace of the sub-namespace.
	// +optional
	SourceParent string `json:"sourceParent,omitempty"`

	// TargetParent is the desired parent namespace of the sub-namespace.
	// +optional
	TargetParent string `json:"targetParent,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:storageversion
//+kubebuilder:subresource:status
//+genclient

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

// MoveSourceParent returns the source parent specified in the move.
func (s *SubNamespace) MoveSourceParent() string {
	if s.Spec.Move == nil {
		return ""
	}
	return s.Spec.Move.SourceParent
}

// MoveTargetParent returns the target parent specified in the move.
func (s *SubNamespace) MoveTargetParent() string {
	if s.Spec.Move == nil {
		return ""
	}
	return s.Spec.Move.TargetParent
}

// IsMoveRequested returns true if this SubNamespace requests a move to another parent.
func (s *SubNamespace) IsMoveRequested() bool {
	return s.Spec.Move != nil &&
		s.Spec.Move.TargetParent != "" &&
		s.Spec.Move.TargetParent != s.Namespace
}

// IsAcceptingMove returns true if this SubNamespace accepts a move from another parent.
func (s *SubNamespace) IsAcceptingMove() bool {
	return s.Spec.Move != nil &&
		s.Spec.Move.SourceParent != "" &&
		s.Spec.Move.SourceParent != s.Namespace
}

// AcceptsMoveFrom returns true if this SubNamespace accepts a move from the given parent.
func (s *SubNamespace) AcceptsMoveFrom(sourceParent string) bool {
	return s.IsAcceptingMove() && s.MoveSourceParent() == sourceParent
}

//+kubebuilder:object:root=true

// SubNamespaceList contains a list of SubNamespace
type SubNamespaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SubNamespace `json:"items"`
}

const (
	SubNamespaceTypeMoveStalled string = "MoveStalled"

	SubNamespaceReasonConflict           string = "Conflict"
	SubNamespaceReasonMoveTargetNotFound string = "MoveTargetNotFound"
	SubNamespaceReasonMoveNotAccepted    string = "MoveNotAccepted"
)
