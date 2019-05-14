package v1

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterPhase is the phase of the storageos cluster at a given point in time.
type ClusterPhase string

// Constants for operator defaults values and different phases.
const (
	ClusterPhaseInitial ClusterPhase = ""
	ClusterPhaseRunning              = "Running"

	DefaultNamespace = "storageos"

	DefaultServiceName         = "storageos"
	DefaultServiceType         = "ClusterIP"
	DefaultServiceExternalPort = 5705
	DefaultServiceInternalPort = 5705

	DefaultIngressHostname = "storageos.local"

	DefaultNodeContainerImage                 = "storageos/node:1.2.0"
	DefaultInitContainerImage                 = "storageos/init:0.1"
	CSIv1ClusterDriverRegistrarContainerImage = "quay.io/k8scsi/csi-cluster-driver-registrar:v1.0.1"
	CSIv1NodeDriverRegistrarContainerImage    = "quay.io/k8scsi/csi-node-driver-registrar:v1.0.1"
	CSIv1ExternalProvisionerContainerImage    = "storageos/csi-provisioner:v1.0.1"
	CSIv1ExternalAttacherContainerImage       = "quay.io/k8scsi/csi-attacher:v1.0.1"
	CSIv1LivenessProbeContainerImage          = "quay.io/k8scsi/livenessprobe:v1.0.1"
	CSIv0DriverRegistrarContainerImage        = "quay.io/k8scsi/driver-registrar:v0.4.2"
	CSIv0ExternalProvisionerContainerImage    = "storageos/csi-provisioner:v0.4.2"
	CSIv0ExternalAttacherContainerImage       = "quay.io/k8scsi/csi-attacher:v0.4.2"

	DefaultPluginRegistrationPath = "/var/lib/kubelet/plugins_registry"
	OldPluginRegistrationPath     = "/var/lib/kubelet/plugins"

	DefaultCSIEndpoint                 = "/storageos/csi.sock"
	DefaultCSIRegistrarSocketDir       = "/var/lib/kubelet/device-plugins/"
	DefaultCSIKubeletDir               = "/var/lib/kubelet"
	DefaultCSIPluginDir                = "/storageos/"
	DefaultCSIDeviceDir                = "/dev"
	DefaultCSIRegistrationDir          = DefaultPluginRegistrationPath
	DefaultCSIKubeletRegistrationPath  = "/storageos/csi.sock"
	DefaultCSIDriverRegistrationMode   = "node-register"
	DefaultCSIDriverRequiresAttachment = "true"
	DefaultCSIHelperDeployment         = "statefulset"
)

func getDefaultCSIEndpoint(pluginRegistrationPath string) string {
	return fmt.Sprintf("%s%s%s", "unix:/", pluginRegistrationPath, DefaultCSIEndpoint)
}

func getDefaultCSIPluginDir(pluginRegistrationPath string) string {
	return path.Join(pluginRegistrationPath, DefaultCSIPluginDir)
}

func getDefaultCSIKubeletRegistrationPath(pluginRegistrationPath string) string {
	return path.Join(pluginRegistrationPath, DefaultCSIKubeletRegistrationPath)
}

// StorageOSClusterSpec defines the desired state of StorageOSCluster
type StorageOSClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// Join is the join token used for service discovery.
	Join string `json:"join"`

	// CSI defines the configurations for CSI.
	CSI StorageOSClusterCSI `json:"csi"`

	// ResourceNS is the kubernetes Namespace where storageos resources are
	// provisioned.
	ResourceNS string `json:"namespace"`

	// Service is the Service configuration for the cluster nodes.
	Service StorageOSClusterService `json:"service"`

	// SecretRefName is the name of the secret object that contains all the
	// sensitive cluster configurations.
	SecretRefName string `json:"secretRefName"`

	// SecretRefNamespace is the namespace of the secret reference.
	SecretRefNamespace string `json:"secretRefNamespace"`

	// SharedDir is the shared directory to be used when the kubelet is running
	// in a container.
	// Typically: "/var/lib/kubelet/plugins/kubernetes.io~storageos".
	// If not set, defaults will be used.
	SharedDir string `json:"sharedDir"`

	// Ingress defines the ingress configurations used in the cluster.
	Ingress StorageOSClusterIngress `json:"ingress"`

	// Images defines the various container images used in the cluster.
	Images ContainerImages `json:"images"`

	// KVBackend defines the key-value store backend used in the cluster.
	KVBackend StorageOSClusterKVBackend `json:"kvBackend"`

	// Pause is to pause the operator for the cluster.
	Pause bool `json:"pause"`

	// Debug is to set debug mode of the cluster.
	Debug bool `json:"debug"`

	// NodeSelectorTerms is to set the placement of storageos pods using
	// node affinity requiredDuringSchedulingIgnoredDuringExecution.
	NodeSelectorTerms []corev1.NodeSelectorTerm `json:"nodeSelectorTerms"`

	// Tolerations is to set the placement of storageos pods using
	// pod toleration.
	Tolerations []corev1.Toleration `json:"tolerations"`

	// Resources is to set the resource requirements of the storageos containers.
	Resources corev1.ResourceRequirements `json:"resources"`

	// Disable Pod Fencing.  With StatefulSets, Pods are only re-scheduled if
	// the Pod has been marked as killed.  In practice this means that failover
	// of a StatefulSet pod is a manual operation.
	//
	// By enabling Pod Fencing and setting the `storageos.com/fenced=true` label
	// on a Pod, StorageOS will enable automated Pod failover (by killing the
	// application Pod on the failed node) if the following conditions exist:
	//
	// - Pod fencing has not been explicitly disabled.
	// - StorageOS has determined that the node the Pod is running on is
	//   offline.  StorageOS uses Gossip and TCP checks and will retry for 30
	//   seconds.  At this point all volumes on the failed node are marked
	//   offline (irrespective of whether fencing is enabled) and volume
	//   failover starts.
	// - The Pod has the label `storageos.com/fenced=true` set.
	// - The Pod has at least one StorageOS volume attached.
	// - Each StorageOS volume has at least 1 healthy replica.
	//
	// When Pod Fencing is disabled, StorageOS will not perform any interaction
	// with Kubernetes when it detects that a node has gone offline.
	// Additionally, the Kubernetes permissions required for Fencing will not be
	// added to the StorageOS role.
	DisableFencing bool `json:"disableFencing"`

	// Disable Telemetry.
	DisableTelemetry bool `json:"disableTelemetry"`

	// TLSEtcdSecretRefName is the name of the secret object that contains the
	// etcd TLS certs. This secret is shared with etcd, therefore it's not part
	// of the main storageos secret.
	TLSEtcdSecretRefName string `json:"tlsEtcdSecretRefName"`

	// TLSEtcdSecretRefNamespace is the namespace of the etcd TLS secret object.
	TLSEtcdSecretRefNamespace string `json:"tlsEtcdSecretRefNamespace"`

	// K8sDistro is the name of the Kubernetes distribution where the operator
	// is being deployed.  It should be in the format: `name[-1.0]`, where the
	// version is optional and should only be appended if known.  Suitable names
	// include: `openshift`, `rancher`, `aks`, `gke`, `eks`, or the deployment
	// method if using upstream directly, e.g `minishift` or `kubeadm`.
	//
	// Setting k8sDistro is optional, and will be used to simplify cluster
	// configuration by setting appropriate defaults for the distribution.  The
	// distribution information will also be included in the product telemetry
	// (if enabled), to help focus development efforts.
	K8sDistro string `json:"k8sDistro"`
}

// StorageOSClusterStatus defines the observed state of StorageOSCluster
type StorageOSClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	Phase            ClusterPhase          `json:"phase"`
	NodeHealthStatus map[string]NodeHealth `json:"nodeHealthStatus,omitempty"`
	Nodes            []string              `json:"nodes"`
	Ready            string                `json:"ready"`
	Members          MembersStatus         `json:"members"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageOSCluster is the Schema for the storageosclusters API
// +k8s:openapi-gen=true
type StorageOSCluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   StorageOSClusterSpec   `json:"spec,omitempty"`
	Status StorageOSClusterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageOSClusterList contains a list of StorageOSCluster
type StorageOSClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []StorageOSCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&StorageOSCluster{}, &StorageOSClusterList{})
}

// MembersStatus stores the status details of cluster member nodes.
type MembersStatus struct {
	// Ready are the storageos cluster members that are ready to serve requests.
	// The member names are the same as the node IPs.
	Ready []string `json:"ready,omitempty"`
	// Unready are the storageos cluster nodes not ready to serve requests.
	Unready []string `json:"unready,omitempty"`
}

// GetResourceNS returns the namespace where all the resources should be provisioned.
func (s StorageOSClusterSpec) GetResourceNS() string {
	if s.ResourceNS != "" {
		return s.ResourceNS
	}
	return DefaultNamespace
}

// GetNodeContainerImage returns node container image.
func (s StorageOSClusterSpec) GetNodeContainerImage() string {
	if s.Images.NodeContainer != "" {
		return s.Images.NodeContainer
	}
	return DefaultNodeContainerImage
}

// GetInitContainerImage returns init container image.
func (s StorageOSClusterSpec) GetInitContainerImage() string {
	if s.Images.InitContainer != "" {
		return s.Images.InitContainer
	}
	return DefaultInitContainerImage
}

// GetCSINodeDriverRegistrarImage returns CSI node driver registrar container image.
func (s StorageOSClusterSpec) GetCSINodeDriverRegistrarImage(csiv1 bool) string {
	if s.Images.CSINodeDriverRegistrarContainer != "" {
		return s.Images.CSINodeDriverRegistrarContainer
	}
	if csiv1 {
		return CSIv1NodeDriverRegistrarContainerImage
	}
	return CSIv0DriverRegistrarContainerImage
}

// GetCSIClusterDriverRegistrarImage returns CSI cluster driver registrar
// container image.
func (s StorageOSClusterSpec) GetCSIClusterDriverRegistrarImage() string {
	if s.Images.CSIClusterDriverRegistrarContainer != "" {
		return s.Images.CSIClusterDriverRegistrarContainer
	}
	return CSIv1ClusterDriverRegistrarContainerImage
}

// GetCSIExternalProvisionerImage returns CSI external provisioner container image.
func (s StorageOSClusterSpec) GetCSIExternalProvisionerImage(csiv1 bool) string {
	if s.Images.CSIExternalProvisionerContainer != "" {
		return s.Images.CSIExternalProvisionerContainer
	}
	if csiv1 {
		return CSIv1ExternalProvisionerContainerImage
	}
	return CSIv0ExternalProvisionerContainerImage
}

// GetCSIExternalAttacherImage returns CSI external attacher container image.
func (s StorageOSClusterSpec) GetCSIExternalAttacherImage(csiv1 bool) string {
	if s.Images.CSIExternalAttacherContainer != "" {
		return s.Images.CSIExternalAttacherContainer
	}
	if csiv1 {
		return CSIv1ExternalAttacherContainerImage
	}
	return CSIv0ExternalAttacherContainerImage
}

// GetCSILivenessProbeImage returns CSI liveness probe container image.
func (s StorageOSClusterSpec) GetCSILivenessProbeImage() string {
	if s.Images.CSILivenessProbeContainer != "" {
		return s.Images.CSILivenessProbeContainer
	}
	return CSIv1LivenessProbeContainerImage
}

// GetServiceName returns the service name.
func (s StorageOSClusterSpec) GetServiceName() string {
	if s.Service.Name != "" {
		return s.Service.Name
	}
	return DefaultServiceName
}

// GetServiceType returns the service type.
func (s StorageOSClusterSpec) GetServiceType() string {
	if s.Service.Type != "" {
		return s.Service.Type
	}
	return DefaultServiceType
}

// GetServiceExternalPort returns the service external port.
func (s StorageOSClusterSpec) GetServiceExternalPort() int {
	if s.Service.ExternalPort != 0 {
		return s.Service.ExternalPort
	}
	return DefaultServiceExternalPort
}

// GetServiceInternalPort returns the service internal port.
func (s StorageOSClusterSpec) GetServiceInternalPort() int {
	if s.Service.InternalPort != 0 {
		return s.Service.InternalPort
	}
	return DefaultServiceInternalPort
}

// GetIngressHostname returns the ingress host name.
func (s StorageOSClusterSpec) GetIngressHostname() string {
	if s.Ingress.Hostname != "" {
		return s.Ingress.Hostname
	}
	return DefaultIngressHostname
}

// GetCSIEndpoint returns the CSI unix socket endpoint path.
func (s StorageOSClusterSpec) GetCSIEndpoint(csiv1 bool) string {
	if s.CSI.Endpoint != "" {
		return s.CSI.Endpoint
	}
	if csiv1 {
		return getDefaultCSIEndpoint(DefaultPluginRegistrationPath)
	}
	return getDefaultCSIEndpoint(OldPluginRegistrationPath)
}

// GetCSIRegistrarSocketDir returns the CSI registrar socket dir.
func (s StorageOSClusterSpec) GetCSIRegistrarSocketDir() string {
	if s.CSI.RegistrarSocketDir != "" {
		return s.CSI.RegistrarSocketDir
	}
	return DefaultCSIRegistrarSocketDir
}

// GetCSIKubeletDir returns the Kubelet dir.
func (s StorageOSClusterSpec) GetCSIKubeletDir() string {
	if s.CSI.KubeletDir != "" {
		return s.CSI.KubeletDir
	}
	return DefaultCSIKubeletDir
}

// GetCSIPluginDir returns the CSI plugin dir.
func (s StorageOSClusterSpec) GetCSIPluginDir(csiv1 bool) string {
	if s.CSI.PluginDir != "" {
		return s.CSI.PluginDir
	}
	if csiv1 {
		return getDefaultCSIPluginDir(DefaultPluginRegistrationPath)
	}
	return getDefaultCSIPluginDir(OldPluginRegistrationPath)
}

// GetCSIDeviceDir returns the CSI device dir.
func (s StorageOSClusterSpec) GetCSIDeviceDir() string {
	if s.CSI.DeviceDir != "" {
		return s.CSI.DeviceDir
	}
	return DefaultCSIDeviceDir
}

// GetCSIRegistrationDir returns the CSI registration dir.
func (s StorageOSClusterSpec) GetCSIRegistrationDir(csiv1 bool) string {
	if s.CSI.RegistrationDir != "" {
		return s.CSI.RegistrationDir
	}
	if csiv1 {
		return DefaultCSIRegistrationDir
	}
	// CSI Registration Dir and Plugin Registration Path are the same.
	return OldPluginRegistrationPath
}

// GetCSIKubeletRegistrationPath returns the CSI Kubelet Registration Path.
func (s StorageOSClusterSpec) GetCSIKubeletRegistrationPath(csiv1 bool) string {
	if s.CSI.KubeletRegistrationPath != "" {
		return s.CSI.KubeletRegistrationPath
	}
	if csiv1 {
		return getDefaultCSIKubeletRegistrationPath(DefaultPluginRegistrationPath)
	}
	return getDefaultCSIKubeletRegistrationPath(OldPluginRegistrationPath)
}

// GetCSIDriverRegistrationMode returns the CSI Driver Registration Mode.
func (s StorageOSClusterSpec) GetCSIDriverRegistrationMode() string {
	if s.CSI.DriverRegistrationMode != "" {
		return s.CSI.DriverRegistrationMode
	}
	return DefaultCSIDriverRegistrationMode
}

// GetCSIDriverRequiresAttachment returns the CSI Driver Requires Attachment
func (s StorageOSClusterSpec) GetCSIDriverRequiresAttachment() string {
	if s.CSI.DriverRequiresAttachment != "" {
		return s.CSI.DriverRequiresAttachment
	}
	return DefaultCSIDriverRequiresAttachment
}

// GetCSIVersion returns the CSI Driver version.
func (s StorageOSClusterSpec) GetCSIVersion(csiv1 bool) string {
	if s.CSI.Version != "" {
		return s.CSI.Version
	}
	if csiv1 {
		return "v1"
	}
	return "v0"
}

// GetCSIHelperDeployment returns the CSI helper deployment strategy value.
func (s StorageOSClusterSpec) GetCSIHelperDeployment() string {
	if s.CSI.HelperDeployment != "" {
		return s.CSI.HelperDeployment
	}
	return DefaultCSIHelperDeployment
}

// ContainerImages contains image names of all the containers used by the operator.
type ContainerImages struct {
	NodeContainer                      string `json:"nodeContainer"`
	InitContainer                      string `json:"initContainer"`
	CSINodeDriverRegistrarContainer    string `json:"csiNodeDriverRegistrarContainer"`
	CSIClusterDriverRegistrarContainer string `json:"csiClusterDriverRegistrarContainer"`
	CSIExternalProvisionerContainer    string `json:"csiExternalProvisionerContainer"`
	CSIExternalAttacherContainer       string `json:"csiExternalAttacherContainer"`
	CSILivenessProbeContainer          string `json:"csiLivenessProbeContainer"`
}

// StorageOSClusterCSI contains CSI configurations.
type StorageOSClusterCSI struct {
	Enable                       bool   `json:"enable"`
	Version                      string `json:"version"`
	Endpoint                     string `json:"endpoint"`
	EnableProvisionCreds         bool   `json:"enableProvisionCreds"`
	EnableControllerPublishCreds bool   `json:"enableControllerPublishCreds"`
	EnableNodePublishCreds       bool   `json:"enableNodePublishCreds"`
	RegistrarSocketDir           string `json:"registrarSocketDir"`
	KubeletDir                   string `json:"kubeletDir"`
	PluginDir                    string `json:"pluginDir"`
	DeviceDir                    string `json:"deviceDir"`
	RegistrationDir              string `json:"registrationDir"`
	KubeletRegistrationPath      string `json:"kubeletRegistrationPath"`
	DriverRegistrationMode       string `json:"driverRegisterationMode"`
	DriverRequiresAttachment     string `json:"driverRequiresAttachment"`
	HelperDeployment             string `json:"helperDeployment"`
}

// StorageOSClusterService contains Service configurations.
type StorageOSClusterService struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	ExternalPort int               `json:"externalPort"`
	InternalPort int               `json:"internalPort"`
	Annotations  map[string]string `json:"annotations"`
}

// StorageOSClusterIngress contains Ingress configurations.
type StorageOSClusterIngress struct {
	Enable      bool              `json:"enable"`
	Hostname    string            `json:"hostname"`
	TLS         bool              `json:"tls"`
	Annotations map[string]string `json:"annotations"`
}

// NodeHealth contains health status of a node.
type NodeHealth struct {
	DirectfsInitiator string `json:"directfsInitiator"`
	Director          string `json:"director"`
	KV                string `json:"kv"`
	KVWrite           string `json:"kvWrite"`
	Nats              string `json:"nats"`
	Presentation      string `json:"presentation"`
	Rdb               string `json:"rdb"`
}

// StorageOSClusterKVBackend stores key-value store backend configurations.
type StorageOSClusterKVBackend struct {
	Address string `json:"address"`
	Backend string `json:"backend"`
}
