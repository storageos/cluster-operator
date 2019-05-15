package storageos

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// Pod toleration keys.
	nodeNotReadyTolKey    = "node.kubernetes.io/not-ready"
	nodeUnreachableTolKey = "node.kubernetes.io/unreachable"
)

// createCSIHelper creates CSI helpers based on the cluster configuration.
func (s *Deployment) createCSIHelper() error {
	// Replicas of the CSI helper pod.
	replicas := int32(1)

	// NOTE: StatefulSet is default for backwards compatibility. In the next
	// major release, Deployment will be the default.
	switch s.stos.Spec.GetCSIDeploymentStrategy() {
	case deploymentKind:
		helperDeployment := s.csiHelperDeployment(replicas)
		return s.createOrUpdateObject(helperDeployment)
	default:
		helperStatefulSet := s.csiHelperStatefulSet(replicas)
		return s.createOrUpdateObject(helperStatefulSet)
	}
}

// csiHelperStatefulSet returns a CSI helper StatefulSet object.
func (s Deployment) csiHelperStatefulSet(replicas int32) *appsv1.StatefulSet {
	podLabels := podLabelsForCSIHelpers(s.stos.Name, statefulsetKind)
	statefulset := &appsv1.StatefulSet{
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
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: StatefulsetSA,
					Containers:         s.csiHelperContainers(),
					Volumes:            s.csiHelperVolumes(),
				},
			},
		},
	}

	s.addCommonPodProperties(&statefulset.Spec.Template.Spec)

	return statefulset
}

// csiHelperDeployment returns a CSI helper Deployment object.
func (s Deployment) csiHelperDeployment(replicas int32) *appsv1.Deployment {
	podLabels := podLabelsForCSIHelpers(s.stos.Name, deploymentKind)
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      csiHelperName,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: podLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: CSIHelperSA,
					Containers:         s.csiHelperContainers(),
					Volumes:            s.csiHelperVolumes(),
				},
			},
		},
	}

	s.addCommonPodProperties(&dep.Spec.Template.Spec)

	return dep
}

// addCommonPodProperties adds common pod properties to a given pod spec. The
// common pod properties are common for all the pods that are part of storageos
// deployment, including the CSI helpers pod.
func (s Deployment) addCommonPodProperties(podSpec *corev1.PodSpec) error {
	s.addPodPriorityClass(podSpec)
	s.addNodeAffinity(podSpec)
	if err := s.addTolerations(podSpec); err != nil {
		return err
	}
	addPodTolerationForRecovery(podSpec)
	return nil
}

// addPodTolerationForRecovery adds pod tolerations for cases when a node isn't
// functional. Usually k8s toleration seconds is five minutes. This sets the
// toleration seconds to 30 seconds.
func addPodTolerationForRecovery(podSpec *corev1.PodSpec) {
	tolerationSeconds := int64(30)
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

// csiHelperContainers returns a list of containers that should be part of the
// CSI helper pods.
func (s Deployment) csiHelperContainers() []corev1.Container {
	containers := []corev1.Container{
		{
			Image:           s.stos.Spec.GetCSIExternalProvisionerImage(CSIV1Supported(s.k8sVersion)),
			Name:            "csi-external-provisioner",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{
				"--v=5",
				"--provisioner=storageos",
				"--csi-address=$(ADDRESS)",
			},
			Env: []corev1.EnvVar{
				{
					Name:  addressEnvVar,
					Value: "/csi/csi.sock",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "plugin-dir",
					MountPath: "/csi",
				},
			},
		},
		{
			Image:           s.stos.Spec.GetCSIExternalAttacherImage(CSIV1Supported(s.k8sVersion)),
			Name:            "csi-external-attacher",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{
				"--v=5",
				"--csi-address=$(ADDRESS)",
			},
			Env: []corev1.EnvVar{
				{
					Name:  addressEnvVar,
					Value: "/csi/csi.sock",
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "plugin-dir",
					MountPath: "/csi",
				},
			},
		},
	}

	// CSI v1 requires running CSI driver registrar to register the driver along
	// with the other CSI helpers.
	// CSI v0 requires the driver registrar to be run with the driver instances
	// only.
	if CSIV1Supported(s.k8sVersion) {
		driverReg := corev1.Container{
			Image:           s.stos.Spec.GetCSIClusterDriverRegistrarImage(),
			Name:            "csi-driver-k8s-registrar",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{
				"--v=5",
				"--csi-address=$(ADDRESS)",
				"--pod-info-mount-version=v1",
			},
			Env: []corev1.EnvVar{
				{
					Name:  addressEnvVar,
					Value: "/csi/csi.sock",
				},
				{
					Name: kubeNodeNameEnvVar,
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							APIVersion: "v1",
							FieldPath:  "spec.nodeName",
						},
					},
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "plugin-dir",
					MountPath: "/csi",
				},
			},
		}
		containers = append(containers, driverReg)
	}

	return containers
}

// csiHelperVolumes returns a list of volumes that should be part of the CSI
// helper pods.
func (s Deployment) csiHelperVolumes() []corev1.Volume {
	hostpathDirOrCreate := corev1.HostPathDirectoryOrCreate
	return []corev1.Volume{
		{
			Name: "plugin-dir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: s.stos.Spec.GetCSIPluginDir(CSIV1Supported(s.k8sVersion)),
					Type: &hostpathDirOrCreate,
				},
			},
		},
	}
}

// getCSIHelperStatefulSet returns the CSI helper StatefulSet resource.
func (s Deployment) getCSIHelperStatefulSet(name string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
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

// getCSIHelperDeployment returns the CSI helper Deployment resource.
func (s Deployment) getCSIHelperDeployment(name string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
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

// deleteCSIHelper deletes the CSI helper based on the cluster configuration.
func (s Deployment) deleteCSIHelper() error {
	// The names of CSI helpers are fixed. Using the appropriate names for the
	// different kinds.
	switch s.stos.Spec.GetCSIDeploymentStrategy() {
	case deploymentKind:
		return s.deleteObject(s.getCSIHelperDeployment(csiHelperName))
	default:
		return s.deleteObject(s.getCSIHelperStatefulSet(statefulsetName))
	}
}

// podLabelsForCSIHelpers takes the name of a cluster custom resource and the
// kind of helper, and returns labels for the pods of the helpers.
func podLabelsForCSIHelpers(name, kind string) map[string]string {
	return map[string]string{
		"app":          appName,
		"storageos_cr": name,
		"kind":         kind,
	}
}
