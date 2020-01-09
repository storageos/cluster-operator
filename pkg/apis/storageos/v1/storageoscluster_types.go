package v1

import (
	"fmt"
	"path"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/storageos/cluster-operator/internal/pkg/image"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ClusterPhase is the phase of the storageos cluster at a given point in time.
type ClusterPhase string

// Constants for operator defaults values and different phases.
const (
	ClusterPhaseInitial ClusterPhase = ""
	// A cluster is in running phase when the cluster health is reported
	// healthy, all the StorageOS nodes are ready.
	ClusterPhaseRunning ClusterPhase = "Running"
	// A cluster is in creating phase when the cluster resource provisioning as
	// started
	ClusterPhaseCreating ClusterPhase = "Creating"
	// A cluster is in pending phase when the creation hasn't started. This can
	// happen if there's an existing cluster and the new cluster provisioning is
	// not allowed by the operator.
	ClusterPhasePending ClusterPhase = "Pending"
	// A cluster is in terminating phase when the cluster delete is initiated.
	// The cluster object is waiting for the finalizers to be executed.
	ClusterPhaseTerminating ClusterPhase = "Terminating"

	DefaultNamespace = "storageos"

	DefaultStorageClassName = "fast"

	DefaultServiceName         = "storageos"
	DefaultServiceType         = "ClusterIP"
	DefaultServiceExternalPort = 5705
	DefaultServiceInternalPort = 5705

	DefaultIngressHostname = "storageos.local"

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
	DefaultCSIDeploymentStrategy       = "statefulset"
)

func getDefaultCSIEndpoint(pluginRegistrationPath string) string {
	return fmt.Sprintf("%s%s%s", "unix://", pluginRegistrationPath, DefaultCSIEndpoint)
}

func getDefaultCSIPluginDir(pluginRegistrationPath string) string {
	return path.Join(pluginRegistrationPath, DefaultCSIPluginDir)
}

func getDefaultCSIKubeletRegistrationPath(pluginRegistrationPath string) string {
	return path.Join(pluginRegistrationPath, DefaultCSIKubeletRegistrationPath)
}

// StorageOSClusterSpec defines the desired state of StorageOSCluster
// +k8s:openapi-gen=true
type StorageOSClusterSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// Join is the join token used for service discovery.
	Join string `json:"join,omitempty"`

	// CSI defines the configurations for CSI.
	CSI StorageOSClusterCSI `json:"csi,omitempty"`

	// Namespace is the kubernetes Namespace where storageos resources are
	// provisioned.
	Namespace string `json:"namespace,omitempty"`

	// StorageClassName is the name of default StorageClass created for
	// StorageOS volumes.
	StorageClassName string `json:"storageClassName,omitempty"`

	// Service is the Service configuration for the cluster nodes.
	Service StorageOSClusterService `json:"service,omitempty"`

	// SecretRefName is the name of the secret object that contains all the
	// sensitive cluster configurations.
	SecretRefName string `json:"secretRefName"`

	// SecretRefNamespace is the namespace of the secret reference.
	SecretRefNamespace string `json:"secretRefNamespace"`

	// SharedDir is the shared directory to be used when the kubelet is running
	// in a container.
	// Typically: "/var/lib/kubelet/plugins/kubernetes.io~storageos".
	// If not set, defaults will be used.
	SharedDir string `json:"sharedDir,omitempty"`

	// Ingress defines the ingress configurations used in the cluster.
	Ingress StorageOSClusterIngress `json:"ingress,omitempty"`

	// Images defines the various container images used in the cluster.
	Images ContainerImages `json:"images,omitempty"`

	// KVBackend defines the key-value store backend used in the cluster.
	KVBackend StorageOSClusterKVBackend `json:"kvBackend,omitempty"`

	// Pause is to pause the operator for the cluster.
	Pause bool `json:"pause,omitempty"`

	// Debug is to set debug mode of the cluster.
	Debug bool `json:"debug,omitempty"`

	// NodeSelectorTerms is to set the placement of storageos pods using
	// node affinity requiredDuringSchedulingIgnoredDuringExecution.
	NodeSelectorTerms []corev1.NodeSelectorTerm `json:"nodeSelectorTerms,omitempty"`

	// Tolerations is to set the placement of storageos pods using
	// pod toleration.
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Resources is to set the resource requirements of the storageos containers.
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

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
	DisableFencing bool `json:"disableFencing,omitempty"`

	// Disable Telemetry.
	DisableTelemetry bool `json:"disableTelemetry,omitempty"`

	// Disable TCMU can be set to true to disable the TCMU storage driver.  This
	// is required when there are multiple storage systems running on the same
	// node and you wish to avoid conflicts.  Only one TCMU-based storage system
	// can run on a node at a time.
	//
	// Disabling TCMU will degrade performance.
	DisableTCMU bool `json:"disableTCMU,omitempty"`

	// Force TCMU can be set to true to ensure that TCMU is enabled or
	// cause StorageOS to abort startup.
	//
	// At startup, StorageOS will automatically fallback to non-TCMU mode if
	// another TCMU-based storage system is running on the node.  Since non-TCMU
	// will degrade performance, this may not always be desired.
	ForceTCMU bool `json:"forceTCMU,omitempty"`

	// TLSEtcdSecretRefName is the name of the secret object that contains the
	// etcd TLS certs. This secret is shared with etcd, therefore it's not part
	// of the main storageos secret.
	TLSEtcdSecretRefName string `json:"tlsEtcdSecretRefName,omitempty"`

	// TLSEtcdSecretRefNamespace is the namespace of the etcd TLS secret object.
	TLSEtcdSecretRefNamespace string `json:"tlsEtcdSecretRefNamespace,omitempty"`

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
	K8sDistro string `json:"k8sDistro,omitempty"`

	// Disable StorageOS scheduler extender.
	DisableScheduler bool `json:"disableScheduler,omitempty"`
}

// StorageOSClusterStatus defines the observed state of StorageOSCluster
// +k8s:openapi-gen=true
type StorageOSClusterStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	Phase            ClusterPhase          `json:"phase,omitempty"`
	NodeHealthStatus map[string]NodeHealth `json:"nodeHealthStatus,omitempty"`
	Nodes            []string              `json:"nodes,omitempty"`
	Ready            string                `json:"ready,omitempty"`
	Members          MembersStatus         `json:"members,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// StorageOSCluster is the Schema for the storageosclusters API
// +k8s:openapi-gen=true
// +kubebuilder:printcolumn:name="ready",type="string",JSONPath=".status.ready",description="Ready status of the storageos nodes."
// +kubebuilder:printcolumn:name="status",type="string",JSONPath=".status.phase",description="Status of the whole cluster."
// +kubebuilder:printcolumn:name="age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:resource:path=storageosclusters,shortName=stos
// +kubebuilder:singular=storageoscluster
// +kubebuilder:subresource:status
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
	if s.Namespace != "" {
		return s.Namespace
	}
	return DefaultNamespace
}

// GetStorageClassName returns the name of default StorageClass created with the
// StorageOS cluster.
func (s StorageOSClusterSpec) GetStorageClassName() string {
	if s.StorageClassName != "" {
		return s.StorageClassName
	}
	return DefaultStorageClassName
}

// GetNodeContainerImage returns node container image.
func (s StorageOSClusterSpec) GetNodeContainerImage() string {
	if s.Images.NodeContainer != "" {
		return s.Images.NodeContainer
	}
	return image.GetDefaultImage(image.StorageOSNodeImageEnvVar, image.DefaultNodeContainerImage)
}

// GetInitContainerImage returns init container image.
func (s StorageOSClusterSpec) GetInitContainerImage() string {
	if s.Images.InitContainer != "" {
		return s.Images.InitContainer
	}
	return image.GetDefaultImage(image.StorageOSInitImageEnvVar, image.DefaultInitContainerImage)
}

// GetCSINodeDriverRegistrarImage returns CSI node driver registrar container image.
func (s StorageOSClusterSpec) GetCSINodeDriverRegistrarImage(csiv1 bool) string {
	if s.Images.CSINodeDriverRegistrarContainer != "" {
		return s.Images.CSINodeDriverRegistrarContainer
	}
	if csiv1 {
		return image.GetDefaultImage(image.CSIv1NodeDriverRegistrarImageEnvVar, image.CSIv1NodeDriverRegistrarContainerImage)
	}
	return image.GetDefaultImage(image.CSIv0DriverRegistrarImageEnvVar, image.CSIv0DriverRegistrarContainerImage)
}

// GetCSIClusterDriverRegistrarImage returns CSI cluster driver registrar
// container image.
func (s StorageOSClusterSpec) GetCSIClusterDriverRegistrarImage() string {
	if s.Images.CSIClusterDriverRegistrarContainer != "" {
		return s.Images.CSIClusterDriverRegistrarContainer
	}
	return image.GetDefaultImage(image.CSIv1ClusterDriverRegistrarImageEnvVar, image.CSIv1ClusterDriverRegistrarContainerImage)
}

// GetCSIExternalProvisionerImage returns CSI external provisioner container image.
func (s StorageOSClusterSpec) GetCSIExternalProvisionerImage(csiv1 bool) string {
	if s.Images.CSIExternalProvisionerContainer != "" {
		return s.Images.CSIExternalProvisionerContainer
	}
	if csiv1 {
		return image.GetDefaultImage(image.CSIv1ExternalProvisionerImageEnvVar, image.CSIv1ExternalProvisionerContainerImage)
	}
	return image.GetDefaultImage(image.CSIv0ExternalProvisionerImageEnvVar, image.CSIv0ExternalProvisionerContainerImage)
}

// GetCSIExternalAttacherImage returns CSI external attacher container image.
// CSI v0, CSI v1 on k8s 1.13 and CSI v1 on k8s 1.14+ require different versions
// of external attacher.
func (s StorageOSClusterSpec) GetCSIExternalAttacherImage(csiv1 bool, attacherv2Supported bool) string {
	if s.Images.CSIExternalAttacherContainer != "" {
		return s.Images.CSIExternalAttacherContainer
	}
	if csiv1 {
		if attacherv2Supported {
			return image.GetDefaultImage(image.CSIv1ExternalAttacherv2ImageEnvVar, image.CSIv1ExternalAttacherv2ContainerImage)
		}
		return image.GetDefaultImage(image.CSIv1ExternalAttacherImageEnvVar, image.CSIv1ExternalAttacherContainerImage)
	}
	return image.GetDefaultImage(image.CSIv0ExternalAttacherImageEnvVar, image.CSIv0ExternalAttacherContainerImage)
}

// GetCSILivenessProbeImage returns CSI liveness probe container image.
func (s StorageOSClusterSpec) GetCSILivenessProbeImage() string {
	if s.Images.CSILivenessProbeContainer != "" {
		return s.Images.CSILivenessProbeContainer
	}
	return image.GetDefaultImage(image.CSIv1LivenessProbeImageEnvVar, image.CSIv1LivenessProbeContainerImage)
}

// GetHyperkubeImage returns hyperkube container image for a given k8s version.
// If an image is set explicitly in the cluster configuration, that image is
// returned.
func (s StorageOSClusterSpec) GetHyperkubeImage(k8sVersion string) string {
	if s.Images.HyperkubeContainer != "" {
		return s.Images.HyperkubeContainer
	}

	// NOTE: Hyperkube is not being used anywhere for now. Hyperkube image is
	// not available to be set via environment variable.
	// Add version prefix "v" in the tag.
	return fmt.Sprintf("%s:v%s", image.DefaultHyperkubeContainerRegistry, k8sVersion)
}

// GetKubeSchedulerImage returns kube-scheduler container image for a given k8s
// version. If an image is set explicitly in the cluster configuration, that
// image is returned.
func (s StorageOSClusterSpec) GetKubeSchedulerImage(k8sVersion string) string {
	if s.Images.KubeSchedulerContainer != "" {
		return s.Images.KubeSchedulerContainer
	}

	// Kube-scheduler image is dynamically selected based on the k8s version.
	// We create an image name for a fallback image based on the k8s version.
	// If kube-scheduler image is not specified in the environment variable, the
	// fallback image is used.

	// Add version prefix "v" in the tag.
	fallbackImage := fmt.Sprintf("%s:v%s", image.DefaultKubeSchedulerContainerRegistry, k8sVersion)

	return image.GetDefaultImage(image.KubeSchedulerImageEnvVar, fallbackImage)
}

// GetNFSServerImage returns NFS server container image used as the default
// image in the cluster.
func (s StorageOSClusterSpec) GetNFSServerImage() string {
	if s.Images.NFSContainer != "" {
		return s.Images.NFSContainer
	}
	return image.GetDefaultImage(image.NFSImageEnvVar, image.DefaultNFSContainerImage)
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

// GetCSIDeploymentStrategy returns the CSI helper deployment strategy value.
func (s StorageOSClusterSpec) GetCSIDeploymentStrategy() string {
	if s.CSI.DeploymentStrategy != "" {
		return s.CSI.DeploymentStrategy
	}
	return DefaultCSIDeploymentStrategy
}

// ContainerImages contains image names of all the containers used by the operator.
type ContainerImages struct {
	NodeContainer                      string `json:"nodeContainer,omitempty"`
	InitContainer                      string `json:"initContainer,omitempty"`
	CSINodeDriverRegistrarContainer    string `json:"csiNodeDriverRegistrarContainer,omitempty"`
	CSIClusterDriverRegistrarContainer string `json:"csiClusterDriverRegistrarContainer,omitempty"`
	CSIExternalProvisionerContainer    string `json:"csiExternalProvisionerContainer,omitempty"`
	CSIExternalAttacherContainer       string `json:"csiExternalAttacherContainer,omitempty"`
	CSILivenessProbeContainer          string `json:"csiLivenessProbeContainer,omitempty"`
	HyperkubeContainer                 string `json:"hyperkubeContainer,omitempty"`
	KubeSchedulerContainer             string `json:"kubeSchedulerContainer,omitempty"`
	NFSContainer                       string `json:"nfsContainer,omitempty"`
}

// StorageOSClusterCSI contains CSI configurations.
type StorageOSClusterCSI struct {
	Enable                       bool   `json:"enable,omitempty"`
	Version                      string `json:"version,omitempty"`
	Endpoint                     string `json:"endpoint,omitempty"`
	EnableProvisionCreds         bool   `json:"enableProvisionCreds,omitempty"`
	EnableControllerPublishCreds bool   `json:"enableControllerPublishCreds,omitempty"`
	EnableNodePublishCreds       bool   `json:"enableNodePublishCreds,omitempty"`
	RegistrarSocketDir           string `json:"registrarSocketDir,omitempty"`
	KubeletDir                   string `json:"kubeletDir,omitempty"`
	PluginDir                    string `json:"pluginDir,omitempty"`
	DeviceDir                    string `json:"deviceDir,omitempty"`
	RegistrationDir              string `json:"registrationDir,omitempty"`
	KubeletRegistrationPath      string `json:"kubeletRegistrationPath,omitempty"`
	DriverRegistrationMode       string `json:"driverRegisterationMode,omitempty"`
	DriverRequiresAttachment     string `json:"driverRequiresAttachment,omitempty"`
	DeploymentStrategy           string `json:"deploymentStrategy,omitempty"`
}

// StorageOSClusterService contains Service configurations.
type StorageOSClusterService struct {
	Name         string            `json:"name"`
	Type         string            `json:"type"`
	ExternalPort int               `json:"externalPort,omitempty"`
	InternalPort int               `json:"internalPort,omitempty"`
	Annotations  map[string]string `json:"annotations,omitempty"`
}

// StorageOSClusterIngress contains Ingress configurations.
type StorageOSClusterIngress struct {
	Enable      bool              `json:"enable,omitempty"`
	Hostname    string            `json:"hostname,omitempty"`
	TLS         bool              `json:"tls,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NodeHealth contains health status of a node.
type NodeHealth struct {
	DirectfsInitiator string `json:"directfsInitiator,omitempty"`
	Director          string `json:"director,omitempty"`
	KV                string `json:"kv,omitempty"`
	KVWrite           string `json:"kvWrite,omitempty"`
	Nats              string `json:"nats,omitempty"`
	Presentation      string `json:"presentation,omitempty"`
	Rdb               string `json:"rdb,omitempty"`
}

// StorageOSClusterKVBackend stores key-value store backend configurations.
type StorageOSClusterKVBackend struct {
	Address string `json:"address,omitempty"`
	Backend string `json:"backend,omitempty"`
}
