package storageos

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Deployment) createStatefulSet() error {
	ls := labelsForStatefulSet(s.stos.Name)
	replicas := int32(1)
	hostpathDirOrCreate := corev1.HostPathDirectoryOrCreate

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
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: StatefulsetSA,
					Containers: []corev1.Container{
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
					},
					Volumes: []corev1.Volume{
						{
							Name: "plugin-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: s.stos.Spec.GetCSIPluginDir(CSIV1Supported(s.k8sVersion)),
									Type: &hostpathDirOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}

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

		sset.Spec.Template.Spec.Containers = append(sset.Spec.Template.Spec.Containers, driverReg)
	}

	podSpec := &sset.Spec.Template.Spec

	s.addNodeAffinity(podSpec, s.stos.Spec.NodeSelectorTerms)

	if err := s.addTolerations(podSpec); err != nil {
		return err
	}

	return s.createOrUpdateObject(sset)
}

func (s *Deployment) deleteStatefulSet(name string) error {
	return s.deleteObject(s.getStatefulSet(name))
}

func (s *Deployment) getStatefulSet(name string) *appsv1.StatefulSet {
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
