package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// DerivedSecretSpec defines the desired state of DerivedSecret.
type DerivedSecretSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of DerivedSecret. Edit derivedsecret_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// DerivedSecretStatus defines the observed state of DerivedSecret.
type DerivedSecretStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DerivedSecret is the Schema for the derivedsecrets API.
type DerivedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   DerivedSecretSpec   `json:"spec,omitempty"`
	Status DerivedSecretStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// DerivedSecretList contains a list of DerivedSecret.
type DerivedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DerivedSecret `json:"items"`
}

func init() {
	SchemeBuilder.Register(&DerivedSecret{}, &DerivedSecretList{})
}
