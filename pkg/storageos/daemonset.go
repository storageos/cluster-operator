package storageos

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/storageos/cluster-operator/pkg/util/k8s"
)

const (

	// Hostname is the name we use to refer to a node.
	hostnameEnvVar = "HOSTNAME"

	// First cluster user's username.
	bootstrapUsernameEnvVar = "BOOTSTRAP_USERNAME"
	// First cluster user's password.
	bootstrapPasswordEnvVar = "BOOTSTRAP_PASSWORD"
	// Namespace created on startup
	// TODO: not sure we need/want this if namespaces are created on demand?
	// bootstrapNamespaceEnvVar = "BOOTSTRAP_NAMESPACE"

	advertiseIPEnvVar  = "ADVERTISE_IP"
	addressEnvVar      = "ADDRESS"
	kubeNodeNameEnvVar = "KUBE_NODE_NAME"

	// Operator vars
	daemonSetNameEnvVar      = "DAEMONSET_NAME"
	daemonSetNamespaceEnvVar = "DAEMONSET_NAMESPACE"

	sysAdminCap = "SYS_ADMIN"
	debugVal    = "xdebug"

	// V1 Only
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
)

func (s *Deployment) createDaemonSet() error {
	ls := podLabelsForDaemonSet(s.stos.Name)
	privileged := true
	mountPropagationBidirectional := corev1.MountPropagationBidirectional
	allowPrivilegeEscalation := true
	configMapOptional := false
	configMapFileMode := int32(0600)

	spec := &appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: ls,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: ls,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: DaemonsetSA,
				HostPID:            true,
				HostNetwork:        true,
				DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
				InitContainers: []corev1.Container{
					{
						Name:  "storageos-init",
						Image: s.stos.Spec.GetInitContainerImage(),
						EnvFrom: []corev1.EnvFromSource{
							corev1.EnvFromSource{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configmapName,
									},
									Optional: &configMapOptional,
								},
							},
						},
						Env: []corev1.EnvVar{
							// Environmental variables for the init container to
							// help query the DaemonSet resource and get the
							// current StorageOS node container image.
							{
								Name:  daemonSetNameEnvVar,
								Value: daemonsetName,
							},
							{
								Name:  daemonSetNamespaceEnvVar,
								Value: s.stos.Spec.GetResourceNS(),
							},
						},
						VolumeMounts: []corev1.VolumeMount{
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
							{
								Name:             "state",
								MountPath:        "/var/lib/storageos",
								MountPropagation: &mountPropagationBidirectional,
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
							Capabilities: &corev1.Capabilities{
								Add: []corev1.Capability{"SYS_ADMIN"},
							},
						},
					},
				},
				Containers: []corev1.Container{
					{
						Image: s.stos.Spec.GetNodeContainerImage(),
						Name:  "storageos",
						Args:  []string{"server"},
						Ports: []corev1.ContainerPort{{
							ContainerPort: 5705,
							Name:          "api",
						}},
						EnvFrom: []corev1.EnvFromSource{
							corev1.EnvFromSource{
								ConfigMapRef: &corev1.ConfigMapEnvSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: configmapName,
									},
									Optional: &configMapOptional,
								},
							},
						},
						Env: []corev1.EnvVar{
							{
								Name: hostnameEnvVar,
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "spec.nodeName",
									},
								},
							},
							{
								Name: bootstrapUsernameEnvVar,
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: initSecretName,
										},
										Key: "username",
									},
								},
							},
							{
								Name: bootstrapPasswordEnvVar,
								ValueFrom: &corev1.EnvVarSource{
									SecretKeyRef: &corev1.SecretKeySelector{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: initSecretName,
										},
										Key: "password",
									},
								},
							},
							{
								Name: advertiseIPEnvVar,
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "status.podIP",
									},
								},
							},
						},
						SecurityContext: &corev1.SecurityContext{
							Privileged: &privileged,
							Capabilities: &corev1.Capabilities{
								Add: []corev1.Capability{sysAdminCap},
							},
							AllowPrivilegeEscalation: &allowPrivilegeEscalation,
						},
						VolumeMounts: []corev1.VolumeMount{
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
							{
								Name:      "config",
								MountPath: "/etc/storageos",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "kernel-modules",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/lib/modules",
							},
						},
					},
					{
						Name: "fuse",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/dev/fuse",
							},
						},
					},
					{
						Name: "sys",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/sys",
							},
						},
					},
					{
						Name: "state",
						VolumeSource: corev1.VolumeSource{
							HostPath: &corev1.HostPathVolumeSource{
								Path: "/var/lib/storageos",
							},
						},
					},
					{
						Name: "config",
						VolumeSource: corev1.VolumeSource{
							ConfigMap: &corev1.ConfigMapVolumeSource{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: configmapName,
								},
								DefaultMode: &configMapFileMode,
								Optional:    &configMapOptional,
							},
						},
					},
				},
			},
		},
		// OnDelete update strategy by default.
		UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.OnDeleteDaemonSetStrategyType,
		},
	}

	podSpec := &spec.Template.Spec
	nodeContainer := &podSpec.Containers[0]

	s.addPodPriorityClass(podSpec)

	s.addTLSEtcdCerts(podSpec)

	s.addNodeAffinity(podSpec)

	// TODO: update when V2 supports health endpoint.
	if !s.nodev2 {
		s.addNodeContainerProbes(nodeContainer)
	}

	if err := s.addTolerations(podSpec); err != nil {
		return err
	}

	s.addNodeContainerResources(nodeContainer)

	s.addSharedDir(podSpec)

	s.addCSI(podSpec)

	return s.k8sResourceManager.DaemonSet(daemonsetName, s.stos.Spec.GetResourceNS(), nil, spec).Create()
}

// podLabelsForDaemonSet takes the name of a cluster custom resource and returns
// labels for the pods of StorageOS node DaemonSet.
func podLabelsForDaemonSet(name string) map[string]string {
	// Combine DaemonSet specific labels with the default app labels.
	labels := map[string]string{
		"app":          appName,
		"storageos_cr": name,
		"kind":         daemonsetKind,
	}
	return k8s.AddDefaultAppLabels(name, labels)
}
