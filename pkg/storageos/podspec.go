package storageos

import (
	"fmt"

	"github.com/storageos/cluster-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

const (
	// Name of kube-system namespace.
	kubeSystemNamespace = "kube-system"

	// Name of the critical priority class.
	criticalPriorityClass = "system-node-critical"
)

// addSharedDir adds env var and volumes for shared dir when running kubelet in
// a container.
func (s *Deployment) addSharedDir(podSpec *corev1.PodSpec) {
	mountPropagationBidirectional := corev1.MountPropagationBidirectional
	nodeContainer := &podSpec.Containers[0]

	// If kubelet is running in a container, sharedDir should be set.
	// TODO: c2 defaults to ROOT_DIR+/volumes
	if s.stos.Spec.SharedDir != "" {
		envVar := corev1.EnvVar{
			Name:  deviceDirEnvVar,
			Value: fmt.Sprintf("%s/devices", s.stos.Spec.SharedDir),
		}
		nodeContainer.Env = append(nodeContainer.Env, envVar)

		sharedDir := corev1.Volume{
			Name: "shared",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: s.stos.Spec.SharedDir,
				},
			},
		}
		podSpec.Volumes = append(podSpec.Volumes, sharedDir)

		volMnt := corev1.VolumeMount{
			Name:             "shared",
			MountPath:        s.stos.Spec.SharedDir,
			MountPropagation: &mountPropagationBidirectional,
		}
		nodeContainer.VolumeMounts = append(nodeContainer.VolumeMounts, volMnt)
	}
}

// addCSI adds the CSI env vars, volumes and containers to the provided podSpec.
func (s *Deployment) addCSI(podSpec *corev1.PodSpec) {
	hostpathDirOrCreate := corev1.HostPathDirectoryOrCreate
	hostpathDir := corev1.HostPathDirectory
	mountPropagationBidirectional := corev1.MountPropagationBidirectional

	nodeContainer := &podSpec.Containers[0]

	// Add CSI specific configurations if enabled.
	if s.stos.Spec.CSI.Enable {
		vols := []corev1.Volume{
			{
				Name: "registrar-socket-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIRegistrarSocketDir(),
						Type: &hostpathDirOrCreate,
					},
				},
			},
			{
				Name: "kubelet-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIKubeletDir(),
						Type: &hostpathDir,
					},
				},
			},
			{
				Name: "plugin-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIPluginDir(CSIV1Supported(s.k8sVersion)),
						Type: &hostpathDirOrCreate,
					},
				},
			},
			{
				Name: "device-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIDeviceDir(),
						Type: &hostpathDir,
					},
				},
			},
			{
				Name: "registration-dir",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: s.stos.Spec.GetCSIRegistrationDir(CSIV1Supported(s.k8sVersion)),
						Type: &hostpathDir,
					},
				},
			},
		}

		podSpec.Volumes = append(podSpec.Volumes, vols...)

		volMnts := []corev1.VolumeMount{
			{
				Name:             "kubelet-dir",
				MountPath:        s.stos.Spec.GetCSIKubeletDir(),
				MountPropagation: &mountPropagationBidirectional,
			},
			{
				Name:      "plugin-dir",
				MountPath: s.stos.Spec.GetCSIPluginDir(CSIV1Supported(s.k8sVersion)),
			},
			{
				Name:      "device-dir",
				MountPath: s.stos.Spec.GetCSIDeviceDir(),
			},
		}

		// Append volume mounts to the first container, the only container is the node container, at this point.
		nodeContainer.VolumeMounts = append(nodeContainer.VolumeMounts, volMnts...)

		// V1 passes CSI credentials as env vars.  In V2, CSI credentials are
		// set in the StorageClass.
		envVar := []corev1.EnvVar{}
		if !s.nodev2 {

			// Append CSI Provision Creds env var if enabled.
			if s.stos.Spec.CSI.EnableProvisionCreds {
				envVar = append(
					envVar,
					corev1.EnvVar{
						Name:  csiRequireCredsCreateEnvVar,
						Value: "true",
					},
					corev1.EnvVar{
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
					corev1.EnvVar{
						Name:  csiRequireCredsCtrlPubEnvVar,
						Value: "true",
					},
					corev1.EnvVar{
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
					corev1.EnvVar{
						Name:  csiRequireCredsNodePubEnvVar,
						Value: "true",
					},
					getCSICredsEnvVar(csiNodePubCredsUsernameEnvVar, csiNodePublishSecretName, "username"),
					getCSICredsEnvVar(csiNodePubCredsPasswordEnvVar, csiNodePublishSecretName, "password"),
				)
			}
		}

		// Append env vars to the first container, node container.
		nodeContainer.Env = append(nodeContainer.Env, envVar...)

		driverReg := corev1.Container{
			Image:           s.stos.Spec.GetCSINodeDriverRegistrarImage(CSIV1Supported(s.k8sVersion)),
			Name:            "csi-driver-registrar",
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
				fmt.Sprintf("--kubelet-registration-path=%s", s.stos.Spec.GetCSIKubeletRegistrationPath(CSIV1Supported(s.k8sVersion))))
		}
		podSpec.Containers = append(podSpec.Containers, driverReg)

		if CSIV1Supported(s.k8sVersion) {
			livenessProbe := corev1.Container{
				Image:           s.stos.Spec.GetCSILivenessProbeImage(),
				Name:            "csi-liveness-probe",
				ImagePullPolicy: corev1.PullIfNotPresent,
				Args: []string{
					"--csi-address=$(ADDRESS)",
					"--connection-timeout=3s",
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
			}
			podSpec.Containers = append(podSpec.Containers, livenessProbe)
		}
	}
}

// addNodeAffinity adds node affinity to the given pod spec from the cluster
// spec NodeSelectorLabel.
func (s *Deployment) addNodeAffinity(podSpec *corev1.PodSpec) {
	if len(s.stos.Spec.NodeSelectorTerms) > 0 {
		podSpec.Affinity = &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: s.stos.Spec.NodeSelectorTerms,
			},
		}}
	}
}

// addTolerations adds tolerations to the given pod spec from cluster
// spec Tolerations.
func (s *Deployment) addTolerations(podSpec *corev1.PodSpec) error {
	return util.AddTolerations(podSpec, s.stos.Spec.Tolerations)
}

// addTLSEtcdCerts adds the etcd TLS secret as a secret mount in the given
// podSpec.
func (s *Deployment) addTLSEtcdCerts(podSpec *corev1.PodSpec) {
	if s.stos.Spec.TLSEtcdSecretRefName != "" &&
		s.stos.Spec.TLSEtcdSecretRefNamespace != "" {
		// Create a secret volume and append to podSpec volumes.
		secretVolume := corev1.Volume{
			Name: tlsEtcdCertsVolume,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: TLSEtcdSecretName,
				},
			},
		}
		podSpec.Volumes = append(podSpec.Volumes, secretVolume)

		// Get the node container from podSpec and add the secret volume at a
		// volume mount.
		nodeContainer := &podSpec.Containers[0]
		secretVolumeMount := corev1.VolumeMount{
			Name:      tlsEtcdCertsVolume,
			MountPath: tlsEtcdRootPath,
		}
		nodeContainer.VolumeMounts = append(nodeContainer.VolumeMounts, secretVolumeMount)

		// Env vars pointing to the volumes are set in the ConfigMap.
	}
}

func (s *Deployment) addPodPriorityClass(podSpec *corev1.PodSpec) {
	// Set pod priority to critical only when deployed in kube-system namespace.
	if s.stos.Spec.GetResourceNS() == kubeSystemNamespace {
		podSpec.PriorityClassName = criticalPriorityClass
	}
}
