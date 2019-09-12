package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// StorageOSUpgradeSpec defines the desired state of StorageOSUpgrade
// +k8s:openapi-gen=true
type StorageOSUpgradeSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// NewImage is the new StorageOS node container image.
	NewImage string `json:"newImage"`
}

// StorageOSUpgradeStatus defines the observed state of StorageOSUpgrade
// +k8s:openapi-gen=true
type StorageOSUpgradeStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// Completed is the status of upgrade process.
	Completed bool `json:"completed,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageOSUpgrade is the Schema for the storageosupgrades API
// +k8s:openapi-gen=true
// +kubebuilder:singular=storageosupgrade
// +kubebuilder:subresource:status
type StorageOSUpgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageOSUpgradeSpec   `json:"spec,omitempty"`
	Status StorageOSUpgradeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageOSUpgradeList contains a list of StorageOSUpgrade
type StorageOSUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageOSUpgrade `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageOSUpgrade{}, &StorageOSUpgradeList{})
}
