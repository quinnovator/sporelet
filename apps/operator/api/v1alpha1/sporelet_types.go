package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// SporeletSpec defines the desired state of Sporelet
// Snapshot points at an OCI reference containing Firecracker snapshot artifacts.
type SporeletSpec struct {
	Snapshot string `json:"snapshot,omitempty"`
}

// SporeletStatus defines the observed state of Sporelet
// Phase reports current lifecycle phase.
type SporeletStatus struct {
	// Phase reports current lifecycle phase
	Phase string `json:"phase,omitempty"`
	// Snapshot records the OCI reference last successfully restored
	Snapshot string `json:"snapshot,omitempty"`
	// Conditions detail the status of snapshot operations
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// Sporelet is the Schema for the sporelets API
type Sporelet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SporeletSpec   `json:"spec,omitempty"`
	Status SporeletStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
// SporeletList contains a list of Sporelet

type SporeletList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Sporelet `json:"items"`
}

var (
	GroupVersion  = schema.GroupVersion{Group: "sporelet.ai", Version: "v1alpha1"}
	SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)
	AddToScheme   = SchemeBuilder.AddToScheme
)

const (
	PhasePending   = "Pending"
	PhaseRestoring = "Restoring"
	PhaseReady     = "Ready"
	PhaseError     = "Error"
	PhaseStopped   = "Stopped"
)

const SporeletFinalizer = "sporelet.ai/cleanup"

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Sporelet{},
		&SporeletList{},
	)
	metav1.AddToGroupVersion(scheme, GroupVersion)
	return nil
}

func init() {
	_ = SchemeBuilder.AddToScheme
}
