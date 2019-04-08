package storageos

import (
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// Names of the storageos daemonset resources.
	daemonsetName            = "storageos-daemonset"
	computeOnlyDaemonsetName = "storageos-compute-only"
)

// createDaemonSet creates storageos storage daemonset.
func (s *Deployment) createDaemonSet() error {
	dset, err := s.getBasicDaemonSetConfiguration(daemonsetName)
	if err != nil {
		return err
	}

	s.addNodeAffinity(&dset.Spec.Template.Spec, s.stos.Spec.NodeSelectorTerms)

	return s.createOrUpdateObject(dset)
}

// createComputeOnlyDaemonSet creates storageos compute only daemonset.
func (s *Deployment) createComputeOnlyDaemonSet() error {
	// Check if node selector terms for compute only is specified.
	if len(s.stos.Spec.ComputeOnlyNodeSelectorTerms) < 1 {
		return nil
	}

	dset, err := s.getBasicDaemonSetConfiguration(computeOnlyDaemonsetName)
	if err != nil {
		return err
	}

	podSpec := &dset.Spec.Template.Spec
	nodeContainer := &podSpec.Containers[0]

	// Pass compute-only label.
	nodeContainer.Env = s.addStorageOSLabelsEnvVars(nodeContainer.Env, computeOnlyLabelVal)

	s.addNodeAffinity(podSpec, s.stos.Spec.ComputeOnlyNodeSelectorTerms)

	return s.createOrUpdateObject(dset)
}

// getBasicDaemonSet creates a basic daemonset configuration for storageos.
func (s *Deployment) getBasicDaemonSetConfiguration(name string) (*appsv1.DaemonSet, error) {
	ls := labelsForDaemonSet(s.stos.Name)
	privileged := true
	mountPropagationBidirectional := corev1.MountPropagationBidirectional
	allowPrivilegeEscalation := true

	dset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Spec: appsv1.DaemonSetSpec{
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
							Name:  "enable-lio",
							Image: s.stos.Spec.GetInitContainerImage(),
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
							LivenessProbe: &corev1.Probe{
								InitialDelaySeconds: int32(65),
								TimeoutSeconds:      int32(10),
								FailureThreshold:    int32(5),
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/v1/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
									},
								},
							},
							ReadinessProbe: &corev1.Probe{
								InitialDelaySeconds: int32(65),
								TimeoutSeconds:      int32(10),
								FailureThreshold:    int32(5),
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path: "/v1/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
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
									Name: adminUsernameEnvVar,
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
									Name: adminPasswordEnvVar,
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
									Name:  joinEnvVar,
									Value: s.stos.Spec.Join,
								},
								{
									Name: advertiseIPEnvVar,
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  namespaceEnvVar,
									Value: s.stos.Spec.GetResourceNS(),
								},
								{
									Name:  disableTelemetryEnvVar,
									Value: strconv.FormatBool(s.stos.Spec.DisableTelemetry),
								},
								{
									Name:  csiVersionEnvVar,
									Value: s.stos.Spec.GetCSIVersion(CSIV1Supported(s.k8sVersion)),
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
					},
				},
			},
		},
	}

	podSpec := &dset.Spec.Template.Spec
	nodeContainer := &podSpec.Containers[0]

	if err := s.addTolerations(podSpec); err != nil {
		return nil, err
	}

	nodeContainer.Env = s.addKVBackendEnvVars(nodeContainer.Env)

	nodeContainer.Env = s.addDebugEnvVars(nodeContainer.Env)

	s.addNodeContainerResources(nodeContainer)

	s.addSharedDir(podSpec)

	s.addCSI(podSpec)

	return dset, nil
}

func (s *Deployment) deleteDaemonSet(name string) error {
	return s.deleteObject(s.getDaemonSet(name))
}

func (s *Deployment) getDaemonSet(name string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
	}
}
