package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterPhase string

const (
	ClusterPhaseInitial ClusterPhase = ""
	ClusterPhaseRunning              = "Running"

	DefaultNamespace = "storageos"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type StorageOSList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []StorageOS `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type StorageOS struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              StorageOSSpec          `json:"spec"`
	Status            StorageOSServiceStatus `json:"status,omitempty"`
}

type StorageOSSpec struct {
	Join       string           `json:"join"`
	EnableCSI  bool             `json:"enableCSI"`
	API        StorageOSAPI     `json:"api"`
	ResourceNS string           `json:"namespace"`
	Service    StorageOSService `json:"service"`
}

// GetResourceNS returns the namespace where all the resources should be provisioned.
func (s StorageOSSpec) GetResourceNS() string {
	if s.ResourceNS != "" {
		return s.ResourceNS
	}
	return DefaultNamespace
}

type StorageOSAPI struct {
	SecretName      string `json:"secretName"`
	SecretNamespace string `json:"secretNamespace"`
	Address         string `json:"address"`
	Username        string `json:"username"`
	Password        string `json:"password"`
}

type StorageOSService struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	ExternalPort int    `json:"externalPort"`
	InternalPort int    `json:"internalPort"`
}

type StorageOSServiceStatus struct {
	Phase ClusterPhase `json:"phase"`
	// ServiceName      string                `json:"serviceName,omitempty"`
	// ClientPort       int                   `json:"clientPort,omitempty"`
	NodeHealthStatus map[string]NodeHealth `json:"nodeHealthStatus,omitempty"`
	Nodes            []string              `json:"nodes"`
	Ready            string                `json:"ready"`
}

type NodeHealth struct {
	DirectfsInitiator string `json:"directfsInitiator"`
	Director          string `json:"director"`
	KV                string `json:"kv"`
	KVWrite           string `json:"kvWrite"`
	Nats              string `json:"nats"`
	Presentation      string `json:"presentation"`
	Rdb               string `json:"rdb"`
	Scheduler         string `json:"scheduler"`
}
