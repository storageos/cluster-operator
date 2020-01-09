package storageos

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/storageos/cluster-operator/pkg/util/k8s/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

const (
	// SchedulerExtenderName is the name of StorageOS scheduler.
	SchedulerExtenderName = "storageos-scheduler"
	// IntreeProvisionerName is the name of the k8s native provisioner.
	IntreeProvisionerName = "kubernetes.io/storageos"
	// CSIProvisionerName is the name of the CSI provisioner.
	CSIProvisionerName = "storageos"
)

const (
	initSecretName                 = "init-secret"
	tlsSecretName                  = "tls-secret"
	csiProvisionerSecretName       = "csi-provisioner-secret"
	csiControllerPublishSecretName = "csi-controller-publish-secret"
	csiNodePublishSecretName       = "csi-node-publish-secret"

	appName         = "storageos"
	daemonsetKind   = "daemonset"
	statefulsetKind = "statefulset"
	deploymentKind  = "deployment"

	daemonsetName   = "storageos-daemonset"
	statefulsetName = "storageos-statefulset"
	configmapName   = "storageos-node-config"
	csiHelperName   = "storageos-csi-helper"

	tlsSecretType       = "kubernetes.io/tls"
	storageosSecretType = "kubernetes.io/storageos"

	defaultFSType                            = "ext4"
	secretNamespaceKey                       = "adminSecretNamespace"
	secretNameKey                            = "adminSecretName"
	apiAddressKey                            = "apiAddress"
	apiUsernameKey                           = "apiUsername"
	apiPasswordKey                           = "apiPassword"
	csiParameterPrefix                       = "csi.storage.k8s.io/"
	csiProvisionUsernameKey                  = "csiProvisionUsername"
	csiProvisionPasswordKey                  = "csiProvisionPassword"
	csiControllerPublishUsernameKey          = "csiControllerPublishUsername"
	csiControllerPublishPasswordKey          = "csiControllerPublishPassword"
	csiNodePublishUsernameKey                = "csiNodePublishUsername"
	csiNodePublishPasswordKey                = "csiNodePublishPassword"
	fsType                                   = "fsType"
	csiV0ProvisionerSecretNameKey            = "csiProvisionerSecretName"
	csiV0ProvisionerSecretNamespaceKey       = "csiProvisionerSecretNamespace"
	csiV0ControllerPublishSecretNameKey      = "csiControllerPublishSecretName"
	csiV0ControllerPublishSecretNamespaceKey = "csiControllerPublishSecretNamespace"
	csiV0NodePublishSecretNameKey            = "csiNodePublishSecretName"
	csiV0NodePublishSecretNamespaceKey       = "csiNodePublishSecretNamespace"
	csiV1FSType                              = csiParameterPrefix + "fstype"
	csiV1ProvisionerSecretNameKey            = csiParameterPrefix + "provisioner-secret-name"
	csiV1ProvisionerSecretNamespaceKey       = csiParameterPrefix + "provisioner-secret-namespace"
	csiV1ControllerPublishSecretNameKey      = csiParameterPrefix + "controller-publish-secret-name"
	csiV1ControllerPublishSecretNamespaceKey = csiParameterPrefix + "controller-publish-secret-namespace"
	csiV1NodePublishSecretNameKey            = csiParameterPrefix + "node-publish-secret-name"
	csiV1NodePublishSecretNamespaceKey       = csiParameterPrefix + "node-publish-secret-namespace"
	tlsCertKey                               = "tls.crt"
	tlsKeyKey                                = "tls.key"
	credUsernameKey                          = "username"
	credPasswordKey                          = "password"

	defaultUsername = "storageos"
	defaultPassword = "storageos"

	// k8s distribution vendor specific keywords.

	// K8SDistroOpenShift is k8s distribution name for OpenShift.
	K8SDistroOpenShift = "openshift"

	// podTolerationSeconds is the time for which a pod tolerates an unfavorable
	// node condition.
	podTolerationSeconds = 30
)

var log = logf.Log.WithName("storageos.cluster")

// Deploy deploys storageos by creating all the resources needed to run storageos.
func (s *Deployment) Deploy() error {
	if err := s.createNamespace(); err != nil {
		return err
	}

	if err := s.createServiceAccountForDaemonSet(); err != nil {
		return err
	}

	if err := s.createRoleForKeyMgmt(); err != nil {
		return err
	}

	if err := s.createRoleBindingForKeyMgmt(); err != nil {
		return err
	}

	if err := s.createClusterRoleForNFS(); err != nil {
		return err
	}

	if err := s.createClusterRoleBindingForNFS(); err != nil {
		return err
	}

	if err := s.createClusterRoleForInit(); err != nil {
		return err
	}

	if err := s.createClusterRoleBindingForInit(); err != nil {
		return err
	}

	if err := s.createInitSecret(); err != nil {
		return err
	}

	if err := s.createTLSEtcdSecret(); err != nil {
		return err
	}

	if err := s.createConfigMap(); err != nil {
		return err
	}

	if err := s.createDaemonSet(); err != nil {
		return err
	}

	if err := s.createService(); err != nil {
		return err
	}

	if s.stos.Spec.Ingress.Enable {
		if s.stos.Spec.Ingress.TLS {
			if err := s.createTLSSecret(); err != nil {
				return err
			}
		}

		if err := s.createIngress(); err != nil {
			return err
		}
	}

	if s.stos.Spec.CSI.Enable {
		// Create CSIDriver if supported.
		supportsCSIDriver, err := HasCSIDriverKind(s.discoveryClient)
		if err != nil {
			return err
		}
		if supportsCSIDriver {
			if err := s.createCSIDriver(); err != nil {
				return err
			}
		}

		// Create CSI exclusive resources.
		if err := s.createCSISecrets(); err != nil {
			return err
		}

		if err := s.createClusterRoleForDriverRegistrar(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForDriverRegistrar(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForK8SDriverRegistrar(); err != nil {
			return err
		}

		if err := s.createServiceAccountForCSIHelper(); err != nil {
			return err
		}

		if err := s.createClusterRoleForProvisioner(); err != nil {
			return err
		}

		if err := s.createClusterRoleForAttacher(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForProvisioner(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForAttacher(); err != nil {
			return err
		}

		if err := s.createCSIHelper(); err != nil {
			return err
		}
	}

	if !s.stos.Spec.DisableScheduler {
		if err := s.createSchedulerExtender(); err != nil {
			return err
		}
	}

	// Add openshift security context constraints.
	if strings.Contains(s.stos.Spec.K8sDistro, K8SDistroOpenShift) {
		if err := s.createClusterRoleForSCC(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForSCC(); err != nil {
			return err
		}
	}

	// Create role for Pod Fencing.
	if !s.stos.Spec.DisableFencing {
		if err := s.createClusterRoleForFencing(); err != nil {
			return err
		}
		if err := s.createClusterRoleBindingForFencing(); err != nil {
			return err
		}
	}

	if err := s.createStorageClass(); err != nil {
		return err
	}

	status, err := s.getStorageOSStatus()
	if err != nil {
		return fmt.Errorf("failed to get storageos status: %v", err)
	}
	return s.updateStorageOSStatus(status)
}

func (s *Deployment) createNamespace() error {
	ns := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
	}

	return resource.CreateOrUpdate(s.client, ns)
}

// addNodeContainerResources adds resource requirements for the node containers.
func (s *Deployment) addNodeContainerResources(nodeContainer *corev1.Container) {
	if s.stos.Spec.Resources.Limits != nil ||
		s.stos.Spec.Resources.Requests != nil {
		nodeContainer.Resources = corev1.ResourceRequirements{
			Limits:   corev1.ResourceList{},
			Requests: corev1.ResourceList{},
		}
		s.stos.Spec.Resources.DeepCopyInto(&nodeContainer.Resources)
	}
}

// addNodeContainerProbes adds probes for the node container healthchecks.
// NOTE: This can be added back into the main DaemonSet once it no longer needs
// to be conditional on the node container version.
func (s *Deployment) addNodeContainerProbes(nodeContainer *corev1.Container) {
	nodeContainer.LivenessProbe = &corev1.Probe{
		InitialDelaySeconds: int32(65),
		TimeoutSeconds:      int32(10),
		FailureThreshold:    int32(5),
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/v1/health",
				Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
			},
		},
	}
	nodeContainer.LivenessProbe = &corev1.Probe{
		InitialDelaySeconds: int32(65),
		TimeoutSeconds:      int32(10),
		FailureThreshold:    int32(5),
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Path: "/v1/health",
				Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
			},
		},
	}
}

// kubeletPluginsWatcherSupported checks if the given version of k8s supports
// KubeletPluginsWatcher. This is used to change the CSI driver registry setup
// based on the kubernetes cluster setup.
func kubeletPluginsWatcherSupported(version string) bool {
	// Supported if v1.12.0 or above.
	return versionSupported(version, "1.12.0")
}

// CSIV1Supported returns true for k8s versions that support CSI v1.
func CSIV1Supported(version string) bool {
	return versionSupported(version, "1.13.0")
}

// CSIExternalAttacherV2Supported returns true for k8s 1.14+
func CSIExternalAttacherV2Supported(version string) bool {
	return versionSupported(version, "1.14.0")
}

// versionSupported takes two versions, current version (haveVersion) and a
// minimum requirement version (wantVersion) and checks if the current version
// is supported by comparing it with the minimum requirement.
func versionSupported(haveVersion, wantVersion string) bool {
	supportedVersion, err := semver.Parse(wantVersion)
	if err != nil {
		log.Info("Failed to parse version", "error", err, "want", wantVersion)
		return false
	}

	currentVersion, err := semver.Parse(haveVersion)
	if err != nil {
		log.Info("Failed to parse version", "error", err, "have", haveVersion)
		return false
	}

	if currentVersion.Compare(supportedVersion) >= 0 {
		return true
	}
	return false
}

// getCSICredsEnvVar returns a corev1.EnvVar object with value from a secret key
// reference, given env var name, reference secret name and key in the secret.
func getCSICredsEnvVar(envVarName, secretName, key string) corev1.EnvVar {
	return corev1.EnvVar{
		Name: envVarName,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}

func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

func asOwner(m *storageosv1.StorageOSCluster) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

func podList() *corev1.PodList {
	return &corev1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// NodeList returns an empty NodeList object.
func NodeList() *corev1.NodeList {
	return &corev1.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
	}
}

func getPodNames(pods []corev1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// GetNodeIPs returns a slice of IPs, given a slice of nodes.
func GetNodeIPs(nodes []corev1.Node) []string {
	ips := []string{}
	for _, node := range nodes {
		// Prefer InternalIP
		if internalIP := GetNodeInternalIP(node.Status.Addresses); internalIP != "" {
			ips = append(ips, internalIP)
			continue
		}
		// Otherwise use first in list.
		if address := GetFirstAddress(node.Status.Addresses); address != "" {
			ips = append(ips, address)
			continue
		}
	}
	return ips
}

// GetNodeInternalIP the InternaIP from a slice of addresses, if it exists.
func GetNodeInternalIP(addresses []corev1.NodeAddress) string {
	for _, addr := range addresses {
		if addr.Type == corev1.NodeInternalIP {
			return addr.Address
		}
	}
	return ""
}

// GetFirstAddress returns the first address from a slice of addresses.
func GetFirstAddress(addresses []corev1.NodeAddress) string {
	for _, addr := range addresses {
		return addr.Address
	}
	return ""
}

// addPodTolerationForRecovery adds pod tolerations for cases when a node isn't
// functional. Usually k8s toleration seconds is five minutes. This sets the
// toleration seconds to 30 seconds.
func addPodTolerationForRecovery(podSpec *corev1.PodSpec) {
	tolerationSeconds := int64(podTolerationSeconds)
	recoveryTolerations := []corev1.Toleration{
		{
			Effect:            corev1.TaintEffectNoExecute,
			Key:               nodeNotReadyTolKey,
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: &tolerationSeconds,
		},
		{
			Effect:            corev1.TaintEffectNoExecute,
			Key:               nodeUnreachableTolKey,
			Operator:          corev1.TolerationOpExists,
			TolerationSeconds: &tolerationSeconds,
		},
	}
	podSpec.Tolerations = append(podSpec.Tolerations, recoveryTolerations...)
}
