package storageos

import (
	"context"
	"fmt"
	"log"

	"github.com/blang/semver"
	api "github.com/storageos/cluster-operator/pkg/apis/storageos/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

	daemonsetName   = "storageos-daemonset"
	statefulsetName = "storageos-statefulset"

	tlsSecretType       = "kubernetes.io/tls"
	storageosSecretType = "kubernetes.io/storageos"

	intreeProvisionerName = "kubernetes.io/storageos"
	csiProvisionerName    = "storageos"

	hostnameEnvVar                      = "HOSTNAME"
	adminUsernameEnvVar                 = "ADMIN_USERNAME"
	adminPasswordEnvVar                 = "ADMIN_PASSWORD"
	joinEnvVar                          = "JOIN"
	advertiseIPEnvVar                   = "ADVERTISE_IP"
	namespaceEnvVar                     = "NAMESPACE"
	deviceDirEnvVar                     = "DEVICE_DIR"
	csiEndpointEnvVar                   = "CSI_ENDPOINT"
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

	sysAdminCap = "SYS_ADMIN"
	debugVal    = "xdebug"

	secretNamespaceKey                     = "adminSecretNamespace"
	secretNameKey                          = "adminSecretName"
	apiAddressKey                          = "apiAddress"
	apiUsernameKey                         = "apiUsername"
	apiPasswordKey                         = "apiPassword"
	csiProvisionUsernameKey                = "csiProvisionUsername"
	csiProvisionPasswordKey                = "csiProvisionPassword"
	csiControllerPublishUsernameKey        = "csiControllerPublishUsername"
	csiControllerPublishPasswordKey        = "csiControllerPublishPassword"
	csiNodePublishUsernameKey              = "csiNodePublishUsername"
	csiNodePublishPasswordKey              = "csiNodePublishPassword"
	csiProvisionerSecretNameKey            = "csiProvisionerSecretName"
	csiProvisionerSecretNamespaceKey       = "csiProvisionerSecretNamespace"
	csiControllerPublishSecretNameKey      = "csiControllerPublishSecretName"
	csiControllerPublishSecretNamespaceKey = "csiControllerPublishSecretNamespace"
	csiNodePublishSecretNameKey            = "csiNodePublishSecretName"
	csiNodePublishSecretNamespaceKey       = "csiNodePublishSecretNamespace"
	tlsCertKey                             = "tls.crt"
	tlsKeyKey                              = "tls.key"

	defaultUsername = "storageos"
	defaultPassword = "storageos"
)

// Deployment stores all the resource configuration and performs
// resource creation and update.
type Deployment struct {
	client     client.Client
	stos       *api.StorageOSCluster
	recorder   record.EventRecorder
	k8sVersion string
	scheme     *runtime.Scheme
	update     bool
}

// NewDeployment creates a new Deployment given a k8c client, storageos manifest
// and an event broadcast recorder.
func NewDeployment(client client.Client, stos *api.StorageOSCluster, recorder record.EventRecorder, scheme *runtime.Scheme, version string, update bool) *Deployment {
	return &Deployment{
		client:     client,
		stos:       stos,
		recorder:   recorder,
		k8sVersion: version,
		scheme:     scheme,
		update:     update,
	}
}

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
	ns := &v1.Namespace{
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

	controllerutil.SetControllerReference(s.stos, ns, s.scheme)
	return s.createOrUpdateObject(ns)
}

func (s *Deployment) createServiceAccount(name string) error {
	sa := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
	}

	controllerutil.SetControllerReference(s.stos, sa, s.scheme)
	return s.createOrUpdateObject(sa)
}

func (s *Deployment) createServiceAccountForDaemonSet() error {
	return s.createServiceAccount("storageos-daemonset-sa")
}

func (s *Deployment) createServiceAccountForStatefulSet() error {
	return s.createServiceAccount("storageos-statefulset-sa")
}

func (s *Deployment) createRoleForKeyMgmt() error {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "key-management-role",
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "create", "delete"},
			},
		},
	}

	controllerutil.SetControllerReference(s.stos, role, s.scheme)
	return s.createOrUpdateObject(role)
}

func (s *Deployment) createClusterRole(name string, rules []rbacv1.PolicyRule) error {
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Rules: rules,
	}

	controllerutil.SetControllerReference(s.stos, role, s.scheme)
	return s.createOrUpdateObject(role)
}

func (s *Deployment) createClusterRoleForDriverRegistrar() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "update"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	return s.createClusterRole("driver-registrar-role", rules)
}

func (s *Deployment) createClusterRoleForProvisioner() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"list", "watch", "create", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"get", "list", "watch", "update"},
		},
		{
			APIGroups: []string{"storageo.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"list", "watch", "get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	return s.createClusterRole("csi-provisioner-role", rules)
}

func (s *Deployment) createClusterRoleForAttacher() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"get", "list", "watch", "update"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"list", "watch", "get"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"volumeattachments"},
			Verbs:     []string{"get", "list", "watch", "update"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	return s.createClusterRole("csi-attacher-role", rules)
}

func (s *Deployment) createRoleBindingForKeyMgmt() error {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "key-management-binding",
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "storageos-daemonset-sa",
				Namespace: s.stos.Spec.GetResourceNS(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "key-management-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	controllerutil.SetControllerReference(s.stos, roleBinding, s.scheme)
	return s.createOrUpdateObject(roleBinding)
}

func (s *Deployment) createClusterRoleBinding(name string, subjects []rbacv1.Subject, roleRef rbacv1.RoleRef) error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": appName,
			},
		},
		Subjects: subjects,
		RoleRef:  roleRef,
	}

	controllerutil.SetControllerReference(s.stos, roleBinding, s.scheme)
	return s.createOrUpdateObject(roleBinding)
}

func (s *Deployment) createClusterRoleBindingForDriverRegistrar() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-daemonset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "driver-registrar-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createClusterRoleBinding("driver-registrar-binding", subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForProvisioner() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-statefulset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "csi-provisioner-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createClusterRoleBinding("csi-provisioner-binding", subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForAttacher() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-statefulset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "csi-attacher-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createClusterRoleBinding("csi-attacher-binding", subjects, roleRef)
}

func (s *Deployment) createDaemonSet() error {
	ls := labelsForDaemonSet(s.stos.Name)
	privileged := true
	mountPropagationBidirectional := v1.MountPropagationBidirectional
	allowPrivilegeEscalation := true

	dset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonsetName,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "storageos-daemonset-sa",
					HostPID:            true,
					HostNetwork:        true,
					DNSPolicy:          v1.DNSClusterFirstWithHostNet,
					InitContainers: []v1.Container{
						{
							Name:  "enable-lio",
							Image: s.stos.Spec.GetInitContainerImage(),
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "kernel-modules",
									MountPath: "/lib/modules",
									ReadOnly:  true,
								},
								{
									Name:             "sys",
									MountPath:        "/sys",
									MountPropagation: &mountPropagationBidirectional,
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &v1.Capabilities{
									Add: []v1.Capability{"SYS_ADMIN"},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Image: s.stos.Spec.GetNodeContainerImage(),
							Name:  "storageos",
							Args:  []string{"server"},
							Ports: []v1.ContainerPort{{
								ContainerPort: 5705,
								Name:          "api",
							}},
							LivenessProbe: &v1.Probe{
								InitialDelaySeconds: int32(65),
								TimeoutSeconds:      int32(10),
								FailureThreshold:    int32(5),
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/v1/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
									},
								},
							},
							ReadinessProbe: &v1.Probe{
								InitialDelaySeconds: int32(65),
								TimeoutSeconds:      int32(10),
								FailureThreshold:    int32(5),
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/v1/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
									},
								},
							},
							Env: []v1.EnvVar{
								{
									Name: hostnameEnvVar,
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name: adminUsernameEnvVar,
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: initSecretName,
											},
											Key: "username",
										},
									},
								},
								{
									Name: adminPasswordEnvVar,
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: initSecretName,
											},
											Key: "password",
										},
									},
								},
								{
									Name:  joinEnvVar,
									Value: s.stos.Spec.Join,
									// ValueFrom: &v1.EnvVarSource{
									// 	FieldRef: &v1.ObjectFieldSelector{
									// 		FieldPath: "status.podIP",
									// 	},
									// },
								},
								{
									Name: advertiseIPEnvVar,
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  namespaceEnvVar,
									Value: s.stos.Spec.GetResourceNS(),
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &v1.Capabilities{
									Add: []v1.Capability{sysAdminCap},
								},
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "fuse",
									MountPath: "/dev/fuse",
								},
								{
									Name:      "sys",
									MountPath: "/sys",
								},
								{
									Name:             "state",
									MountPath:        "/var/lib/storageos",
									MountPropagation: &mountPropagationBidirectional,
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "kernel-modules",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/lib/modules",
								},
							},
						},
						{
							Name: "fuse",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/dev/fuse",
								},
							},
						},
						{
							Name: "sys",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
						{
							Name: "state",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/storageos",
								},
							},
						},
					},
				},
			},
		},
	}

	podSpec := &dset.Spec.Template.Spec
	nodeContainer := &podSpec.Containers[0]

	s.addNodeAffinity(podSpec)

	nodeContainer.Env = s.addKVBackendEnvVars(nodeContainer.Env)

	nodeContainer.Env = s.addDebugEnvVars(nodeContainer.Env)

	s.addNodeContainerResources(nodeContainer)

	s.addSharedDir(podSpec)

	s.addCSI(podSpec)

	controllerutil.SetControllerReference(s.stos, dset, s.scheme)
	return s.createOrUpdateObject(dset)
}

// addNodeContainerResources adds resource requirements for the node containers.
func (s *Deployment) addNodeContainerResources(nodeContainer *v1.Container) {
	if s.stos.Spec.Resources.Limits != nil ||
		s.stos.Spec.Resources.Requests != nil {
		nodeContainer.Resources = v1.ResourceRequirements{
			Limits:   v1.ResourceList{},
			Requests: v1.ResourceList{},
		}
		s.stos.Spec.Resources.DeepCopyInto(&nodeContainer.Resources)
	}
}

// kubeletPluginsWatcherSupported checks if the given version of k8s supports
// KubeletPluginsWatcher. This is used to change the CSI driver registry setup
// based on the kubernetes cluster setup.
func kubeletPluginsWatcherSupported(version string) bool {
	supportedVersion, err := semver.Parse("1.12.0")
	if err != nil {
		log.Printf("failed to parse version: %v", err)
		return false
	}

	currentVersion, err := semver.Parse(version)
	if err != nil {
		log.Printf("failed to parse version: %v", err)
		return false
	}

	// Supported if v1.12.0 or above.
	if currentVersion.Compare(supportedVersion) >= 0 {
		return true
	}
	return false
}

// addKVBackendEnvVars checks if KVBackend is set and sets the appropriate env vars.
func (s *Deployment) addKVBackendEnvVars(env []v1.EnvVar) []v1.EnvVar {
	kvStoreEnv := []v1.EnvVar{}
	if s.stos.Spec.KVBackend.Address != "" {
		kvAddressEnv := v1.EnvVar{
			Name:  kvAddrEnvVar,
			Value: s.stos.Spec.KVBackend.Address,
		}
		kvStoreEnv = append(kvStoreEnv, kvAddressEnv)
	}

	if s.stos.Spec.KVBackend.Backend != "" {
		kvBackendEnv := v1.EnvVar{
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
func (s *Deployment) addDebugEnvVars(env []v1.EnvVar) []v1.EnvVar {
	if s.stos.Spec.Debug {
		debugEnvVar := v1.EnvVar{
			Name:  debugEnvVar,
			Value: debugVal,
		}
		return append(env, debugEnvVar)
	}
	return env
}

// addSharedDir adds env var and volumes for shared dir when running kubelet in
// a container.
func (s *Deployment) addSharedDir(podSpec *v1.PodSpec) {
	mountPropagationBidirectional := v1.MountPropagationBidirectional
	nodeContainer := &podSpec.Containers[0]

	// If kubelet is running in a container, sharedDir should be set.
	if s.stos.Spec.SharedDir != "" {
		envVar := v1.EnvVar{
			Name:  deviceDirEnvVar,
			Value: fmt.Sprintf("%s/devices", s.stos.Spec.SharedDir),
		}
		nodeContainer.Env = append(nodeContainer.Env, envVar)

		sharedDir := v1.Volume{
			Name: "shared",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: s.stos.Spec.SharedDir,
				},
			},
		}
		podSpec.Volumes = append(podSpec.Volumes, sharedDir)

		volMnt := v1.VolumeMount{
			Name:             "shared",
			MountPath:        s.stos.Spec.SharedDir,
			MountPropagation: &mountPropagationBidirectional,
		}
		nodeContainer.VolumeMounts = append(nodeContainer.VolumeMounts, volMnt)
	}
}

// addCSI adds the CSI env vars, volumes and containers to the provided podSpec.
func (s *Deployment) addCSI(podSpec *v1.PodSpec) {
	hostpathDirOrCreate := v1.HostPathDirectoryOrCreate
	hostpathDir := v1.HostPathDirectory
	mountPropagationBidirectional := v1.MountPropagationBidirectional

	nodeContainer := &podSpec.Containers[0]

	// Add CSI specific configurations if enabled.
	if s.stos.Spec.CSI.Enable {
		vols := []v1.Volume{
			{
				Name: "registrar-socket-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIRegistrarSocketDir(),
						Type: &hostpathDirOrCreate,
					},
				},
			},
			{
				Name: "kubelet-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIKubeletDir(),
						Type: &hostpathDir,
					},
				},
			},
			{
				Name: "plugin-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIPluginDir(),
						Type: &hostpathDirOrCreate,
					},
				},
			},
			{
				Name: "device-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIDeviceDir(),
						Type: &hostpathDir,
					},
				},
			},
			{
				Name: "registration-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIRegistrationDir(),
						Type: &hostpathDir,
					},
				},
			},
		}

		podSpec.Volumes = append(podSpec.Volumes, vols...)

		volMnts := []v1.VolumeMount{
			{
				Name:             "kubelet-dir",
				MountPath:        s.stos.Spec.GetCSIKubeletDir(),
				MountPropagation: &mountPropagationBidirectional,
			},
			{
				Name:      "plugin-dir",
				MountPath: s.stos.Spec.GetCSIPluginDir(),
			},
			{
				Name:      "device-dir",
				MountPath: s.stos.Spec.GetCSIDeviceDir(),
			},
		}

		// Append volume mounts to the first container, the only container is the node container, at this point.
		nodeContainer.VolumeMounts = append(nodeContainer.VolumeMounts, volMnts...)

		envVar := []v1.EnvVar{
			{
				Name:  csiEndpointEnvVar,
				Value: s.stos.Spec.GetCSIEndpoint(),
			},
		}

		// Append CSI Provision Creds env var if enabled.
		if s.stos.Spec.CSI.EnableProvisionCreds {
			envVar = append(
				envVar,
				v1.EnvVar{
					Name:  csiRequireCredsCreateEnvVar,
					Value: "true",
				},
				v1.EnvVar{
					Name:  csiRequireCredsDeleteEnvVar,
					Value: "true",
				},
				getCSICredsEnvVar(csiProvisionCredsUsernameEnvVar, csiProvisionerSecretName, "username"),
				getCSICredsEnvVar(csiProvisionCredsPasswordEnvVar, csiProvisionerSecretName, "password"),
			)
		}

		// Append CSI Controller Publish env var if enabled.
		if s.stos.Spec.CSI.EnableControllerPublishCreds {
			envVar = append(
				envVar,
				v1.EnvVar{
					Name:  csiRequireCredsCtrlPubEnvVar,
					Value: "true",
				},
				v1.EnvVar{
					Name:  csiRequireCredsCtrlUnpubEnvVar,
					Value: "true",
				},
				getCSICredsEnvVar(csiControllerPubCredsUsernameEnvVar, csiControllerPublishSecretName, "username"),
				getCSICredsEnvVar(csiControllerPubCredsPasswordEnvVar, csiControllerPublishSecretName, "password"),
			)
		}

		// Append CSI Node Publish env var if enabled.
		if s.stos.Spec.CSI.EnableNodePublishCreds {
			envVar = append(
				envVar,
				v1.EnvVar{
					Name:  csiRequireCredsNodePubEnvVar,
					Value: "true",
				},
				getCSICredsEnvVar(csiNodePubCredsUsernameEnvVar, csiNodePublishSecretName, "username"),
				getCSICredsEnvVar(csiNodePubCredsPasswordEnvVar, csiNodePublishSecretName, "password"),
			)
		}

		// Append env vars to the first container, node container.
		nodeContainer.Env = append(nodeContainer.Env, envVar...)

		driverReg := v1.Container{
			Image:           s.stos.Spec.GetCSIDriverRegistrarImage(),
			Name:            "csi-driver-registrar",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args: []string{
				"--v=5",
				"--csi-address=$(ADDRESS)",
			},
			Env: []v1.EnvVar{
				{
					Name:  addressEnvVar,
					Value: "/csi/csi.sock",
				},
				{
					Name: kubeNodeNameEnvVar,
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "spec.nodeName",
						},
					},
				},
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "plugin-dir",
					MountPath: "/csi",
				},
				{
					Name:      "registrar-socket-dir",
					MountPath: "/var/lib/csi/sockets/",
				},
				{
					Name:      "registration-dir",
					MountPath: "/registration",
				},
			},
		}

		// Add extra flags to activate node-register mode if kubelet plugins
		// watcher is supported.
		if kubeletPluginsWatcherSupported(s.k8sVersion) {
			driverReg.Args = append(
				driverReg.Args,
				"--mode=node-register",
				"--driver-requires-attachment=true",
				"--pod-info-mount-version=v1",
				"--kubelet-registration-path=/var/lib/kubelet/plugins/storageos/csi.sock")
		}
		podSpec.Containers = append(podSpec.Containers, driverReg)
	}
}

// addNodeAffinity adds node affinity to the given pod spec from the cluster
// spec NodeSelectorLabel.
func (s *Deployment) addNodeAffinity(podSpec *v1.PodSpec) {
	if len(s.stos.Spec.NodeSelectorTerms) > 0 {
		podSpec.Affinity = &v1.Affinity{NodeAffinity: &v1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &v1.NodeSelector{
				NodeSelectorTerms: s.stos.Spec.NodeSelectorTerms,
			},
		}}
	}
}

// getCSICredsEnvVar returns a v1.EnvVar object with value from a secret key
// reference, given env var name, reference secret name and key in the secret.
func getCSICredsEnvVar(envVarName, secretName, key string) v1.EnvVar {
	return v1.EnvVar{
		Name: envVarName,
		ValueFrom: &v1.EnvVarSource{
			SecretKeyRef: &v1.SecretKeySelector{
				LocalObjectReference: v1.LocalObjectReference{
					Name: secretName,
				},
				Key: key,
			},
		},
	}
}

func (s *Deployment) createStatefulSet() error {
	ls := labelsForStatefulSet(s.stos.Name)
	replicas := int32(1)
	hostpathDirOrCreate := v1.HostPathDirectoryOrCreate

	sset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      statefulsetName,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "storageos",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "storageos-statefulset-sa",
					Containers: []v1.Container{
						{
							Image:           s.stos.Spec.GetCSIExternalProvisionerImage(),
							Name:            "csi-external-provisioner",
							ImagePullPolicy: v1.PullIfNotPresent,
							Args: []string{
								"--v=5",
								"--provisioner=storageos",
								"--csi-address=$(ADDRESS)",
							},
							Env: []v1.EnvVar{
								{
									Name:  addressEnvVar,
									Value: "/csi/csi.sock",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "plugin-dir",
									MountPath: "/csi",
								},
							},
						},
						{
							Image:           s.stos.Spec.GetCSIExternalAttacherImage(),
							Name:            "csi-external-attacher",
							ImagePullPolicy: v1.PullIfNotPresent,
							Args: []string{
								"--v=5",
								"--csi-address=$(ADDRESS)",
							},
							Env: []v1.EnvVar{
								{
									Name:  addressEnvVar,
									Value: "/csi/csi.sock",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "plugin-dir",
									MountPath: "/csi",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "plugin-dir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: s.stos.Spec.GetCSIPluginDir(),
									Type: &hostpathDirOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}

	controllerutil.SetControllerReference(s.stos, sset, s.scheme)
	return s.createOrUpdateObject(sset)
}

func (s *Deployment) createService() error {
	svc := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.stos.Spec.GetServiceName(),
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
			Annotations: s.stos.Spec.Service.Annotations,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceType(s.stos.Spec.GetServiceType()),
			Ports: []v1.ServicePort{
				{
					Name:       s.stos.Spec.GetServiceName(),
					Protocol:   "TCP",
					Port:       int32(s.stos.Spec.GetServiceInternalPort()),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(s.stos.Spec.GetServiceExternalPort())},
				},
			},
			Selector: map[string]string{
				"app":  appName,
				"kind": daemonsetKind,
			},
		},
	}

	controllerutil.SetControllerReference(s.stos, svc, s.scheme)
	if err := s.createOrUpdateObject(svc); err != nil {
		return err
	}

	// Patch storageos-api secret with above service IP in apiAddress.
	if !s.stos.Spec.CSI.Enable {
		secret := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.stos.Spec.SecretRefName,
				Namespace: s.stos.Spec.SecretRefNamespace,
			},
		}
		nsNameSecret := types.NamespacedName{
			Namespace: secret.ObjectMeta.GetNamespace(),
			Name:      secret.ObjectMeta.GetName(),
		}
		if err := s.client.Get(context.Background(), nsNameSecret, secret); err != nil {
			return err
		}

		nsNameService := types.NamespacedName{
			Namespace: svc.ObjectMeta.GetNamespace(),
			Name:      svc.ObjectMeta.GetName(),
		}
		if err := s.client.Get(context.Background(), nsNameService, svc); err != nil {
			return err
		}

		apiAddress := fmt.Sprintf("tcp://%s:5705", svc.Spec.ClusterIP)
		secret.Data[apiAddressKey] = []byte(apiAddress)

		if err := s.client.Update(context.Background(), secret); err != nil {
			return err
		}
	}

	return nil
}

func (s *Deployment) createIngress() error {
	ingress := &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-ingress",
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
			Annotations: s.stos.Spec.Ingress.Annotations,
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: s.stos.Spec.GetServiceName(),
				ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(s.stos.Spec.GetServiceExternalPort())},
			},
		},
	}

	if s.stos.Spec.Ingress.TLS {
		ingress.Spec.TLS = []v1beta1.IngressTLS{
			v1beta1.IngressTLS{
				Hosts:      []string{s.stos.Spec.Ingress.Hostname},
				SecretName: tlsSecretName,
			},
		}
	}

	controllerutil.SetControllerReference(s.stos, ingress, s.scheme)
	return s.createOrUpdateObject(ingress)
}

func (s *Deployment) createTLSSecret() error {
	cert, key, err := s.getTLSData()
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsSecretName,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
		Type: v1.SecretType(tlsSecretType),
		Data: map[string][]byte{
			tlsCertKey: cert,
			tlsKeyKey:  key,
		},
	}

	controllerutil.SetControllerReference(s.stos, secret, s.scheme)
	return s.createOrUpdateObject(secret)
}

func (s *Deployment) createInitSecret() error {
	username, password, err := s.getAdminCreds()
	if err != nil {
		return err
	}
	if err := s.createCredSecret(initSecretName, username, password); err != nil {
		return err
	}
	return nil
}

func (s *Deployment) getAdminCreds() ([]byte, []byte, error) {
	var username, password []byte
	if s.stos.Spec.SecretRefName != "" && s.stos.Spec.SecretRefNamespace != "" {
		se := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.stos.Spec.SecretRefName,
				Namespace: s.stos.Spec.SecretRefNamespace,
			},
		}
		nsName := types.NamespacedName{
			Name:      se.ObjectMeta.GetName(),
			Namespace: se.ObjectMeta.GetNamespace(),
		}
		if err := s.client.Get(context.Background(), nsName, se); err != nil {
			return nil, nil, err
		}

		username = se.Data[apiUsernameKey]
		password = se.Data[apiPasswordKey]
	} else {
		// Use the default credentials.
		username = []byte(defaultUsername)
		password = []byte(defaultPassword)
	}

	return username, password, nil
}

func (s *Deployment) getTLSData() ([]byte, []byte, error) {
	var cert, key []byte
	if s.stos.Spec.SecretRefName != "" && s.stos.Spec.SecretRefNamespace != "" {
		se := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.stos.Spec.SecretRefName,
				Namespace: s.stos.Spec.SecretRefNamespace,
			},
		}
		nsName := types.NamespacedName{
			Name:      se.ObjectMeta.GetName(),
			Namespace: se.ObjectMeta.GetNamespace(),
		}
		if err := s.client.Get(context.Background(), nsName, se); err != nil {
			return nil, nil, err
		}

		cert = se.Data[tlsCertKey]
		key = se.Data[tlsKeyKey]
	} else {
		cert = []byte("")
		key = []byte("")
	}

	return cert, key, nil
}

// createCSISecrets checks which CSI creds are enabled and creates secret for
// those components.
func (s *Deployment) createCSISecrets() error {
	// Create Provision Secret.
	if s.stos.Spec.CSI.EnableProvisionCreds {
		username, password, err := s.getCSICreds(csiProvisionUsernameKey, csiProvisionPasswordKey)
		if err != nil {
			return err
		}
		if err := s.createCredSecret(csiProvisionerSecretName, username, password); err != nil {
			return err
		}
	}

	// Create Controller Publish Secret.
	if s.stos.Spec.CSI.EnableControllerPublishCreds {
		username, password, err := s.getCSICreds(csiControllerPublishUsernameKey, csiControllerPublishPasswordKey)
		if err != nil {
			return err
		}
		if err := s.createCredSecret(csiControllerPublishSecretName, username, password); err != nil {
			return err
		}
	}

	// Create Node Publish Secret.
	if s.stos.Spec.CSI.EnableNodePublishCreds {
		username, password, err := s.getCSICreds(csiNodePublishUsernameKey, csiNodePublishPasswordKey)
		if err != nil {
			return err
		}
		if err := s.createCredSecret(csiNodePublishSecretName, username, password); err != nil {
			return err
		}
	}

	return nil
}

func (s *Deployment) createCredSecret(name string, username, password []byte) error {
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
		Type: v1.SecretType(v1.SecretTypeOpaque),
		Data: map[string][]byte{
			"username": username,
			"password": password,
		},
	}

	controllerutil.SetControllerReference(s.stos, secret, s.scheme)
	return s.createOrUpdateObject(secret)
}

// getCSICreds - given username and password keys, it fetches the creds from
// storageos-api secret and returns them.
func (s *Deployment) getCSICreds(usernameKey, passwordKey string) (username []byte, password []byte, err error) {
	// Get the username and password from storageos-api secret object.
	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.stos.Spec.SecretRefName,
			Namespace: s.stos.Spec.SecretRefNamespace,
		},
	}
	nsName := types.NamespacedName{
		Name:      secret.ObjectMeta.GetName(),
		Namespace: secret.ObjectMeta.GetNamespace(),
	}
	if err := s.client.Get(context.Background(), nsName, secret); err != nil {
		return nil, nil, err
	}

	username = secret.Data[usernameKey]
	password = secret.Data[passwordKey]

	return username, password, err
}

func (s *Deployment) createStorageClass() error {
	// Provisioner name for in-tree storage plugin.
	provisioner := intreeProvisionerName

	if s.stos.Spec.CSI.Enable {
		provisioner = csiProvisionerName
	}

	sc := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "fast",
			Labels: map[string]string{
				"app": appName,
			},
		},
		Provisioner: provisioner,
		Parameters: map[string]string{
			"pool":   "default",
			"fsType": "ext4",
		},
	}

	if s.stos.Spec.CSI.Enable {
		// Add CSI creds secrets in parameters.
		if s.stos.Spec.CSI.EnableProvisionCreds {
			sc.Parameters[csiProvisionerSecretNameKey] = csiProvisionerSecretName
			sc.Parameters[csiProvisionerSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
		}
		if s.stos.Spec.CSI.EnableControllerPublishCreds {
			sc.Parameters[csiControllerPublishSecretNameKey] = csiControllerPublishSecretName
			sc.Parameters[csiControllerPublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
		}
		if s.stos.Spec.CSI.EnableNodePublishCreds {
			sc.Parameters[csiNodePublishSecretNameKey] = csiNodePublishSecretName
			sc.Parameters[csiNodePublishSecretNamespaceKey] = s.stos.Spec.GetResourceNS()
		}
	} else {
		// Add StorageOS admin secrets name and namespace.
		sc.Parameters[secretNamespaceKey] = s.stos.Spec.SecretRefNamespace
		sc.Parameters[secretNameKey] = s.stos.Spec.SecretRefName
	}

	controllerutil.SetControllerReference(s.stos, sc, s.scheme)
	return s.createOrUpdateObject(sc)
}

// createOrUpdateObject attempts to create a given object. If the object already
// exists and `Deployment.update` is false, no change is made. If update is true,
// the existing object is updated.
func (s *Deployment) createOrUpdateObject(obj runtime.Object) error {
	if err := s.client.Create(context.Background(), obj); err != nil {
		if apierrors.IsAlreadyExists(err) && s.update {
			return s.client.Update(context.Background(), obj)
		} else if !apierrors.IsAlreadyExists(err) {
			kind := obj.GetObjectKind().GroupVersionKind().Kind
			return fmt.Errorf("failed to create %s: %v", kind, err)
		}
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

func asOwner(m *api.StorageOSCluster) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

func podList() *v1.PodList {
	return &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

// NodeList returns an empty NodeList object.
func NodeList() *v1.NodeList {
	return &v1.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
			APIVersion: "v1",
		},
	}
}

func getPodNames(pods []v1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

// GetNodeIPs returns a slice of IPs, given a slice of nodes.
func GetNodeIPs(nodes []v1.Node) []string {
	var ips []string
	for _, node := range nodes {
		ips = append(ips, node.Status.Addresses[0].Address)
	}
	return ips
}
