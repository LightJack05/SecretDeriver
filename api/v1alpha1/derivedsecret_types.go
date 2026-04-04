package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// DerivedSecretSpec defines the desired state of DerivedSecret.
type DerivedSecretSpec struct {
	// ParentSecretRef is a reference to the Kubernetes Secret whose value will be used as
	// the HKDF input key material. The secret may be in any namespace; cross-namespace
	// references require the parent secret to carry the
	// secretderiver.lightjack.de/allowCrossnamespaceReference=true label.
	// +kubebuilder:validation:Required
	ParentSecretRef corev1.SecretReference `json:"parentSecretRef"`

	// ParentSecretKey is the key within the parent secret whose value is used as the
	// HKDF input key material.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	ParentSecretKey string `json:"parentSecretKey"`

	// GeneratedSecretKey is the key name written into the generated Kubernetes Secret.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:MinLength=1
	GeneratedSecretKey string `json:"generatedSecretKey"`
}

// DerivedSecretStatus defines the observed state of DerivedSecret.
type DerivedSecretStatus struct {
	// Conditions contains the latest reconciliation conditions for this DerivedSecret.
	// +operator-sdk:csv:customresourcedefinitions:type=status
	Conditions []metav1.Condition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,1,rep,name=conditions"`

	// Phase is a short summary of the current reconciliation state.
	// It is either "Ready" or "Error".
	// +optional
	Phase string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// DerivedSecret instructs the SecretDeriver operator to deterministically derive a
// Kubernetes Secret from a parent secret using HKDF-SHA256. The operator creates a
// Secret with the same name and namespace as the DerivedSecret, containing the derived
// value at the key specified by GeneratedSecretKey.
//
// The HKDF salt is set to "namespace/name" of this resource, ensuring each DerivedSecret
// produces a unique value even when multiple resources share the same parent secret and key.
type DerivedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// +kubebuilder:validation:Required
	Spec   DerivedSecretSpec   `json:"spec"`
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
