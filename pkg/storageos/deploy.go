package storageos

import (
	"fmt"
	"strings"

	"github.com/blang/semver"
	"github.com/storageos/cluster-operator/pkg/util/k8s/resource"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	// SchedulerExtenderName is the name of StorageOS scheduler.
	SchedulerExtenderName = "storageos-scheduler"
	// IntreeProvisionerName is the name of the k8s native provisioner.
	IntreeProvisionerName = "kubernetes.io/storageos"
	// CSIProvisionerName is the name of the CSI provisioner.
	CSIProvisionerName = "storageos"
	// StorageOSProvisionerName is the new CSI provisioner name.
	StorageOSProvisionerName = "csi.storageos.com"
)

const (
	initSecretName                 = "init-secret"
	tlsSecretName                  = "tls-secret"
	csiProvisionerSecretName       = "csi-provisioner-secret"
	csiControllerPublishSecretName = "csi-controller-publish-secret"
	csiNodePublishSecretName       = "csi-node-publish-secret"
	csiControllerExpandSecretName  = "csi-controller-expand-secret"

	appName         = "storageos"
	daemonsetKind   = "daemonset"
	statefulsetKind = "statefulset"
	deploymentKind  = "deployment"

	daemonsetName   = "storageos-daemonset"
	statefulsetName = "storageos-statefulset"
	configmapName   = "storageos-node-config"
	csiHelperName   = "storageos-csi-helper"

	defaultFSType                          = "ext4"
	apiAddressKey                          = "apiAddress"
	apiUsernameKey                         = "apiUsername"
	apiPasswordKey                         = "apiPassword"
	csiParameterPrefix                     = "csi.storage.k8s.io/"
	csiProvisionUsernameKey                = "csiProvisionUsername"
	csiProvisionPasswordKey                = "csiProvisionPassword"
	csiControllerPublishUsernameKey        = "csiControllerPublishUsername"
	csiControllerPublishPasswordKey        = "csiControllerPublishPassword"
	csiNodePublishUsernameKey              = "csiNodePublishUsername"
	csiNodePublishPasswordKey              = "csiNodePublishPassword"
	csiControllerExpandUsernameKey         = "csiControllerExpandUsername"
	csiControllerExpandPasswordKey         = "csiControllerExpandPassword"
	csiFSType                              = csiParameterPrefix + "fstype"
	csiProvisionerSecretNameKey            = csiParameterPrefix + "provisioner-secret-name"
	csiProvisionerSecretNamespaceKey       = csiParameterPrefix + "provisioner-secret-namespace"
	csiControllerPublishSecretNameKey      = csiParameterPrefix + "controller-publish-secret-name"
	csiControllerPublishSecretNamespaceKey = csiParameterPrefix + "controller-publish-secret-namespace"
	csiNodePublishSecretNameKey            = csiParameterPrefix + "node-publish-secret-name"
	csiNodePublishSecretNamespaceKey       = csiParameterPrefix + "node-publish-secret-namespace"
	csiControllerExpandSecretNameKey       = csiParameterPrefix + "controller-expand-secret-name"
	csiControllerExpandSecretnamespaceKey  = csiParameterPrefix + "controller-expand-secret-namespace"
	tlsCertKey                             = "tls.crt"
	tlsKeyKey                              = "tls.key"
	credUsernameKey                        = "username"
	credPasswordKey                        = "password"

	defaultUsername = "storageos"
	defaultPassword = "storageos"

	// recommendedPidLimit is passed to the init container to warn if a lower
	// limit is set.  It can't be overridden.
	recommendedPidLimit = "32768"

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

		if err := s.createClusterRoleForResizer(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForProvisioner(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForAttacher(); err != nil {
			return err
		}

		if err := s.createClusterRoleBindingForResizer(); err != nil {
			return err
		}

		if err := s.createCSIHelper(); err != nil {
			return err
		}
	}

	if err := s.createAPIManager(); err != nil {
		return err
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

// CSIExternalResizerSupported returns true for k8s 1.16+.
func CSIExternalResizerSupported(version string) bool {
	return versionSupported(version, "1.16.0")
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

// func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
//     obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
// }

// func asOwner(m *storageosv1.StorageOSCluster) metav1.OwnerReference {
//     trueVar := true
//     return metav1.OwnerReference{
//         APIVersion: m.APIVersion,
//         Kind:       m.Kind,
//         Name:       m.Name,
//         UID:        m.UID,
//         Controller: &trueVar,
//     }
// }

// func podList() *corev1.PodList {
//     return &corev1.PodList{
//         TypeMeta: metav1.TypeMeta{
//             Kind:       "Pod",
//             APIVersion: "v1",
//         },
//     }
// }

// NodeList returns an empty NodeList object.
func NodeList() *corev1.NodeList {
	return &corev1.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
	}
}

// func getPodNames(pods []corev1.Pod) []string {
//     var podNames []string
//     for _, pod := range pods {
//         podNames = append(podNames, pod.Name)
//     }
//     return podNames
// }

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
