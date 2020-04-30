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

	s.addCommonPodProperties(&spec.Template.Spec)

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

	s.addCommonPodProperties(&spec.Template.Spec)

	return s.k8sResourceManager.Deployment(csiHelperName, s.stos.Spec.GetResourceNS(), nil, spec).Create()
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

// csiHelperContainers returns a list of containers that should be part of the
// CSI helper pods.
func (s Deployment) csiHelperContainers() ([]corev1.Container, error) {
	privileged := true
	containers := []corev1.Container{
		{
			Image:           s.stos.Spec.GetCSIExternalProvisionerImage(CSIV1Supported(s.k8sVersion), s.nodev2),
			Name:            "csi-external-provisioner",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{
				"--v=5",
				"--provisioner=storageos", // deprecated in v1.1.0, but required for v1.0.0 (CSI v0).
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
		{
			Image:           s.stos.Spec.GetCSIExternalAttacherImage(CSIV1Supported(s.k8sVersion), CSIExternalAttacherV2Supported(s.k8sVersion)),
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

	// v2 provisioner requires additional startup flag.
	if s.nodev2 {
		containers[0].Args = append(containers[0].Args, "--extra-create-metadata")
	}

	// CSI v1 requires running CSI driver registrar to register the driver along
	// with the other CSI helpers.
	// CSI v0 requires the driver registrar to be run with the driver instances
	// only.
	// In k8s 1.13, csi-cluster-driver-registrar was required to be run along
	// with the CSI helpers. This was responsible for the creation of CSIDriver
	// resource belonging to the CRD csidrivers.csi.storage.k8s.io. In k8s
	// 1.14+ this was replaced by a CSIDriver built-in resource belonging to
	// API group csidrivers.storage.k8s.io. This is no longer automatically
	// created. The deployment tools should create this resource.
	//
	// Add csi-cluster-driver-registrar if the built-in csidrivers resource is
	// not supported by the k8s api server.
	supportsCSIDriver, err := HasCSIDriverKind(s.discoveryClient)
	if err != nil {
		return containers, err
	}

	// If CSIDriver is not supported but CSI v1 is supported, run
	// cluster-driver-registrar.
	if !supportsCSIDriver && CSIV1Supported(s.k8sVersion) {
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
		containers = append(containers, driverReg)
	}

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
					Path: s.stos.Spec.GetCSIPluginDir(CSIV1Supported(s.k8sVersion)),
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
		"app":          appName,
		"storageos_cr": name,
		"kind":         kind,
	}
	return k8s.AddDefaultAppLabels(name, labels)
}
