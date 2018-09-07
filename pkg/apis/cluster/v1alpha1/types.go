package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterPhase string

const (
	ClusterPhaseInitial ClusterPhase = ""
	ClusterPhaseRunning              = "Running"

	DefaultNamespace = "storageos"

	DefaultServiceName         = "storageos"
	DefaultServiceType         = "ClusterIP"
	DefaultServiceExternalPort = 5705
	DefaultServiceInternalPort = 5705

	DefaultIngressHostname = "storageos.local"

	DefaultNodeContainerImage                   = "storageos/node:1.0.0-rc4"
	DefaultCSIDriverRegistrarContainerImage     = "quay.io/k8scsi/driver-registrar:v0.2.0"
	DefaultCSIExternalProvisionerContainerImage = "quay.io/k8scsi/csi-provisioner:v0.3.0"
	DefaultCSIExternalAttacherContainerImage    = "quay.io/k8scsi/csi-attacher:v0.3.0"
	DefaultInitContainerImage                   = "storageos/init:0.1"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type StorageOSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`
	Items           []StorageOSCluster `json:"items"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type StorageOSCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              StorageOSSpec          `json:"spec"`
	Status            StorageOSServiceStatus `json:"status,omitempty"`
}

type StorageOSSpec struct {
	Join               string           `json:"join"`
	CSI                StorageOSCSI     `json:"csi"`
	ResourceNS         string           `json:"namespace"`
	Service            StorageOSService `json:"service"`
	SecretRefName      string           `json:"secretRefName"`
	SecretRefNamespace string           `json:"secretRefNamespace"`
	SharedDir          string           `json:"sharedDir"`
	Ingress            StorageOSIngress `json:"ingress"`
	Images             ContainerImages  `json:"images"`
}

// GetResourceNS returns the namespace where all the resources should be provisioned.
func (s StorageOSSpec) GetResourceNS() string {
	if s.ResourceNS != "" {
		return s.ResourceNS
	}
	return DefaultNamespace
}

// GetNodeContainerImage returns node container image.
func (s StorageOSSpec) GetNodeContainerImage() string {
	if s.Images.NodeContainer != "" {
		return s.Images.NodeContainer
	}
	return DefaultNodeContainerImage
}

// GetInitContainerImage returns init container image.
func (s StorageOSSpec) GetInitContainerImage() string {
	if s.Images.InitContainer != "" {
		return s.Images.InitContainer
	}
	return DefaultInitContainerImage
}

// GetCSIDriverRegistrarImage returns CSI driver registrar container image.
func (s StorageOSSpec) GetCSIDriverRegistrarImage() string {
	if s.Images.CSIDriverRegistrarContainer != "" {
		return s.Images.CSIDriverRegistrarContainer
	}
	return DefaultCSIDriverRegistrarContainerImage
}

// GetCSIExternalProvisionerImage returns CSI external provisioner container image.
func (s StorageOSSpec) GetCSIExternalProvisionerImage() string {
	if s.Images.CSIExternalProvisionerContainer != "" {
		return s.Images.CSIExternalProvisionerContainer
	}
	return DefaultCSIExternalProvisionerContainerImage
}

// GetCSIExternalAttacherImage returns CSI external attacher container image.
func (s StorageOSSpec) GetCSIExternalAttacherImage() string {
	if s.Images.CSIExternalAttacherContainer != "" {
		return s.Images.CSIExternalAttacherContainer
	}
	return DefaultCSIExternalAttacherContainerImage
}

// GetServiceName returns the service name.
func (s StorageOSSpec) GetServiceName() string {
	if s.Service.Name != "" {
		return s.Service.Name
	}
	return DefaultServiceName
}

// GetServiceType returns the service type.
func (s StorageOSSpec) GetServiceType() string {
	if s.Service.Type != "" {
		return s.Service.Type
	}
	return DefaultServiceType
}

// GetServiceExternalPort returns the service external port.
func (s StorageOSSpec) GetServiceExternalPort() int {
	if s.Service.ExternalPort != 0 {
		return s.Service.ExternalPort
	}
	return DefaultServiceExternalPort
}

// GetServiceInternalPort returns the service internal port.
func (s StorageOSSpec) GetServiceInternalPort() int {
	if s.Service.InternalPort != 0 {
		return s.Service.InternalPort
	}
	return DefaultServiceInternalPort
}

// GetIngressHostname returns the ingress host name.
func (s StorageOSSpec) GetIngressHostname() string {
	if s.Ingress.Hostname != "" {
		return s.Ingress.Hostname
	}
	return DefaultIngressHostname
}

// ContainerImages contains image names of all the containers used by the operator.
type ContainerImages struct {
	NodeContainer                   string `json:"nodeContainer"`
	InitContainer                   string `json:"initContainer"`
	CSIDriverRegistrarContainer     string `json:"csiDriverRegistrarContainer"`
	CSIExternalProvisionerContainer string `json:"csiExternalProvisionerContainer"`
	CSIExternalAttacherContainer    string `json:"csiExternalAttacherContainer"`
}

type StorageOSCSI struct {
	Enable                       bool `json:"enable"`
	EnableProvisionCreds         bool `json:"enableProvisionCreds"`
	EnableControllerPublishCreds bool `json:"enableControllerPublishCreds"`
	EnableNodePublishCreds       bool `json:"enableNodePublishCreds"`
}

type StorageOSService struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	ExternalPort int               `json:"externalPort"`
	InternalPort int               `json:"internalPort"`
	Annotations  map[string]string `json:"annotations"`
}

type StorageOSIngress struct {
	Enable      bool              `json:"enable"`
	Hostname    string            `json:"hostname"`
	TLS         bool              `json:"tls"`
	Annotations map[string]string `json:"annotations"`
}

type StorageOSServiceStatus struct {
	Phase            ClusterPhase          `json:"phase"`
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
