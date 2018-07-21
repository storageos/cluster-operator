package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterPhase string

const (
	ClusterPhaseInitial ClusterPhase = ""
	ClusterPhaseRunning              = "Running"
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
	Join string `json:"join"`
}

type StorageOSServiceStatus struct {
	Phase            ClusterPhase          `json:"phase"`
	ServiceName      string                `json:"serviceName,omitempty"`
	ClientPort       int                   `json:"clientPort,omitempty"`
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
