package storageos

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/blang/semver"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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

	statefulsetName = "storageos-statefulset"

	tlsSecretType       = "kubernetes.io/tls"
	storageosSecretType = "kubernetes.io/storageos"

	intreeProvisionerName = "kubernetes.io/storageos"
	csiProvisionerName    = "storageos"

	hostnameEnvVar                      = "HOSTNAME"
	adminUsernameEnvVar                 = "ADMIN_USERNAME"
	adminPasswordEnvVar                 = "ADMIN_PASSWORD"
	labelsEnvVar                        = "LABELS"
	joinEnvVar                          = "JOIN"
	advertiseIPEnvVar                   = "ADVERTISE_IP"
	namespaceEnvVar                     = "NAMESPACE"
	disableFencingEnvVar                = "DISABLE_FENCING"
	disableTelemetryEnvVar              = "DISABLE_TELEMETRY"
	deviceDirEnvVar                     = "DEVICE_DIR"
	csiEndpointEnvVar                   = "CSI_ENDPOINT"
	csiVersionEnvVar                    = "CSI_VERSION"
	csiRequireCredsCreateEnvVar         = "CSI_REQUIRE_CREDS_CREATE_VOL"
	csiRequireCredsDeleteEnvVar         = "CSI_REQUIRE_CREDS_DELETE_VOL"
	csiProvisionCredsUsernameEnvVar     = "CSI_PROVISION_CREDS_USERNAME"
	csiProvisionCredsPasswordEnvVar     = "CSI_PROVISION_CREDS_PASSWORD"
	csiRequireCredsCtrlPubEnvVar        = "CSI_REQUIRE_CREDS_CTRL_PUB_VOL"
	csiRequireCredsCtrlUnpubEnvVar      = "CSI_REQUIRE_CREDS_CTRL_UNPUB_VOL"
	csiControllerPubCredsUsernameEnvVar = "CSI_CTRL_PUB_CREDS_USERNAME"
	csiControllerPubCredsPasswordEnvVar = "CSI_CTRL_PUB_CREDS_PASSWORD"
	csiRequireCredsNodePubEnvVar        = "CSI_REQUIRE_CREDS_NODE_PUB_VOL"
	csiNodePubCredsUsernameEnvVar       = "CSI_NODE_PUB_CREDS_USERNAME"
	csiNodePubCredsPasswordEnvVar       = "CSI_NODE_PUB_CREDS_PASSWORD"
	addressEnvVar                       = "ADDRESS"
	kubeNodeNameEnvVar                  = "KUBE_NODE_NAME"
	kvAddrEnvVar                        = "KV_ADDR"
	kvBackendEnvVar                     = "KV_BACKEND"
	debugEnvVar                         = "LOG_LEVEL"

	sysAdminCap         = "SYS_ADMIN"
	debugVal            = "xdebug"
	computeOnlyLabelVal = "storageos.com/deployment=computeonly"

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

	defaultUsername = "storageos"
	defaultPassword = "storageos"
)

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

	if err := s.createInitSecret(); err != nil {
		return err
	}

	// Compute-only daemonset.
	if err := s.createComputeOnlyDaemonSet(); err != nil {
		return err
	}

	// Storage daemonset.
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

		if err := s.createServiceAccountForStatefulSet(); err != nil {
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

		if err := s.createStatefulSet(); err != nil {
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

	return s.createOrUpdateObject(ns)
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

func versionSupported(haveVersion, wantVersion string) bool {
	supportedVersion, err := semver.Parse(wantVersion)
	if err != nil {
		log.Printf("failed to parse version: %v", err)
		return false
	}

	currentVersion, err := semver.Parse(haveVersion)
	if err != nil {
		log.Printf("failed to parse version: %v", err)
		return false
	}

	if currentVersion.Compare(supportedVersion) >= 0 {
		return true
	}
	return false
}

// addKVBackendEnvVars checks if KVBackend is set and sets the appropriate env vars.
func (s *Deployment) addKVBackendEnvVars(env []corev1.EnvVar) []corev1.EnvVar {
	kvStoreEnv := []corev1.EnvVar{}
	if s.stos.Spec.KVBackend.Address != "" {
		kvAddressEnv := corev1.EnvVar{
			Name:  kvAddrEnvVar,
			Value: s.stos.Spec.KVBackend.Address,
		}
		kvStoreEnv = append(kvStoreEnv, kvAddressEnv)
	}

	if s.stos.Spec.KVBackend.Backend != "" {
		kvBackendEnv := corev1.EnvVar{
			Name:  kvBackendEnvVar,
			Value: s.stos.Spec.KVBackend.Backend,
		}
		kvStoreEnv = append(kvStoreEnv, kvBackendEnv)
	}

	if len(kvStoreEnv) > 0 {
		return append(env, kvStoreEnv...)
	}
	return env
}

// addDebugEnvVars checks if the debug mode is set and set the appropriate env var.
func (s *Deployment) addDebugEnvVars(env []corev1.EnvVar) []corev1.EnvVar {
	if s.stos.Spec.Debug {
		debugEnvVar := corev1.EnvVar{
			Name:  debugEnvVar,
			Value: debugVal,
		}
		return append(env, debugEnvVar)
	}
	return env
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

// addStorageOSLabelsEnvVars checks if the debug mode is set and set the appropriate env var.
func (s Deployment) addStorageOSLabelsEnvVars(env []corev1.EnvVar, labels string) []corev1.EnvVar {
	// Return the argument env var if no labels are specified.
	if len(labels) == 0 {
		return env
	}

	// Check if labels env var already exists.
	labelsExists := false
	for _, envVar := range env {
		if envVar.Name == labelsEnvVar {
			// Append the label with new label entries.
			// The labels are separated by ",".
			// e.g.: "storageos.com/deployment=computeonly,country=us,env=prod"
			envVar.Value = strings.Join([]string{envVar.Value, labels}, ",")
			labelsExists = true
			break
		}
	}

	// Add new labels env var if it doesn't exists.
	if !labelsExists {
		labelsEnvVar := corev1.EnvVar{
			Name:  labelsEnvVar,
			Value: labels,
		}
		return append(env, labelsEnvVar)
	}
	return env
}

// createOrUpdateObject attempts to create a given object. If the object already
// exists and `Deployment.update` is false, no change is made. If update is true,
// the existing object is updated.
func (s *Deployment) createOrUpdateObject(obj runtime.Object) error {
	if err := s.client.Create(context.Background(), obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			if s.update {
				return s.client.Update(context.Background(), obj)
			}
			// Exists, no update.
			return nil
		}

		kind := obj.GetObjectKind().GroupVersionKind().Kind
		return fmt.Errorf("failed to create %s: %v", kind, err)
	}
	return nil
}

// deleteObject deletes a given runtime object.
func (s *Deployment) deleteObject(obj runtime.Object) error {
	if err := s.client.Delete(context.Background(), obj); err != nil {
		// If not found, the object has already been deleted.
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

func labelsForDaemonSet(name string) map[string]string {
	return map[string]string{"app": appName, "storageos_cr": name, "kind": daemonsetKind}
}

func labelsForStatefulSet(name string) map[string]string {
	return map[string]string{"app": appName, "storageos_cr": name, "kind": statefulsetKind}
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
	var ips []string
	for _, node := range nodes {
		ips = append(ips, node.Status.Addresses[0].Address)
	}
	return ips
}
