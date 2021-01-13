package storageos

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/storageos/cluster-operator/pkg/util/k8s"
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
		return s.createCSIHelperDeployment(replicas)
	default:
		return s.createCSIHelperStatefulSet(replicas)
	}
}

// csiHelperStatefulSet returns a CSI helper StatefulSet object.
func (s Deployment) createCSIHelperStatefulSet(replicas int32) error {
	podLabels := podLabelsForCSIHelpers(s.stos.Name, statefulsetKind)
	containers, err := s.csiHelperContainers()
	if err != nil {
		return err
	}
	spec := &appsv1.StatefulSetSpec{
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
				Containers:         containers,
				Volumes:            s.csiHelperVolumes(),
			},
		},
	}

	if err := s.addCommonPodProperties(&spec.Template.Spec); err != nil {
		return err
	}

	return s.k8sResourceManager.StatefulSet(statefulsetName, s.stos.Spec.GetResourceNS(), nil, spec).Create()
}

// csiHelperDeployment returns a CSI helper Deployment object.
func (s Deployment) createCSIHelperDeployment(replicas int32) error {
	podLabels := podLabelsForCSIHelpers(s.stos.Name, deploymentKind)
	containers, err := s.csiHelperContainers()
	if err != nil {
		return err
	}
	spec := &appsv1.DeploymentSpec{
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
				Containers:         containers,
				Volumes:            s.csiHelperVolumes(),
			},
		},
	}

	if err := s.addCommonPodProperties(&spec.Template.Spec); err != nil {
		return err
	}

	return s.k8sResourceManager.Deployment(csiHelperName, s.stos.Spec.GetResourceNS(), nil, spec).Create()
}

// addCommonPodProperties adds common pod properties to a given pod spec. The
// common pod properties are common for all the pods that are part of storageos
// deployment, including the CSI helpers pod.
func (s Deployment) addCommonPodProperties(podSpec *corev1.PodSpec) error {
	s.addNodeAffinity(podSpec)

	// Add helper tolerations.
	if err := s.addHelperTolerations(podSpec, podTolerationSeconds); err != nil {
		return err
	}
	return nil
}

// csiHelperContainers returns a list of containers that should be part of the
// CSI helper pods.
func (s Deployment) csiHelperContainers() ([]corev1.Container, error) {
	privileged := true
	containers := []corev1.Container{
		{
			Image:           s.stos.Spec.GetCSIExternalProvisionerImage(),
			Name:            "csi-external-provisioner",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{
				"--v=5",
				"--csi-address=$(ADDRESS)",
				"--extra-create-metadata",
			},
			Env: []corev1.EnvVar{
				{
					Name:  addressEnvVar,
					Value: "/csi/csi.sock",
				},
			},
			SecurityContext: &corev1.SecurityContext{
				Privileged: &privileged,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "plugin-dir",
					MountPath: "/csi",
				},
			},
		},
		{
			Image:           s.stos.Spec.GetCSIExternalAttacherImage(),
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
			SecurityContext: &corev1.SecurityContext{
				Privileged: &privileged,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "plugin-dir",
					MountPath: "/csi",
				},
			},
		},
	}

	// v2 supports volume resize.
	resizer := corev1.Container{
		Image:           s.stos.Spec.GetCSIExternalResizerImage(),
		Name:            "csi-external-resizer",
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
		SecurityContext: &corev1.SecurityContext{
			Privileged: &privileged,
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "plugin-dir",
				MountPath: "/csi",
			},
		},
	}
	containers = append(containers, resizer)

	return containers, nil
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
					Path: s.stos.Spec.GetCSIPluginDir(),
					Type: &hostpathDirOrCreate,
				},
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
		return s.k8sResourceManager.Deployment(csiHelperName, s.stos.Spec.GetResourceNS(), nil, nil).Delete()
	default:
		return s.k8sResourceManager.StatefulSet(statefulsetName, s.stos.Spec.GetResourceNS(), nil, nil).Delete()
	}
}

// podLabelsForCSIHelpers takes the name of a cluster custom resource and the
// kind of helper, and returns labels for the pods of the helpers.
func podLabelsForCSIHelpers(name, kind string) map[string]string {
	// Combine CSI Helper specific labels with the default app labels.
	labels := map[string]string{
		"app":            appName,
		"storageos_cr":   name,
		"kind":           kind,
		k8s.AppComponent: csiHelperName,
	}
	return k8s.AddDefaultAppLabels(name, labels)
}
