package nfs

import (
	"github.com/storageos/cluster-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// DataVolName is the NFS data volume name.
	DataVolName = "nfs-data"
)

func (d *Deployment) createStatefulSet(pvcVS *corev1.PersistentVolumeClaimVolumeSource, nfsPort int, httpPort int) error {
	replicas := int32(1)

	spec := &appsv1.StatefulSetSpec{
		ServiceName: d.nfsServer.Name,
		Replicas:    &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: d.labelsForStatefulSet(),
		},
		Template: d.createPodTemplateSpec(nfsPort, httpPort),
	}

	// Add the block volume in the pod spec volumes.
	vol := corev1.Volume{
		Name: DataVolName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: pvcVS,
		},
	}
	spec.Template.Spec.Volumes = append(spec.Template.Spec.Volumes, vol)

	if err := util.AddTolerations(&spec.Template.Spec, d.nfsServer.Spec.Tolerations); err != nil {
		return err
	}

	// If the cluster was configured with node selectors to only run on certain
	// nodes, use the same selectors to selct the nodes that the NFS pods can
	// run on.  NFSServer does not currently allow setting node selectors or
	// affinity directly.
	util.AddRequiredNodeAffinity(&spec.Template.Spec, d.cluster.Spec.NodeSelectorTerms)

	return d.k8sResourceManager.StatefulSet(d.nfsServer.Name, d.nfsServer.Namespace, nil, spec).Create()
}

func (d *Deployment) createPodTemplateSpec(nfsPort int, httpPort int) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: d.labelsForStatefulSet(),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: d.getServiceAccountName(),
			Containers: []corev1.Container{
				{
					ImagePullPolicy: "IfNotPresent",
					Name:            "nfsd",
					Image:           d.nfsServer.Spec.GetContainerImage(d.cluster.Spec.GetNFSServerImage()),
					Env: []corev1.EnvVar{
						{
							Name:  "GANESHA_CONFIGFILE",
							Value: "/config/" + d.nfsServer.Name,
						},
						{
							Name:  "NAME",
							Value: d.nfsServer.Name,
						},
						{
							Name:  "NAMESPACE",
							Value: d.nfsServer.Namespace,
						},
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "nfs-port",
							ContainerPort: int32(nfsPort),
						},
						{
							Name:          "http-port",
							ContainerPort: int32(httpPort),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "nfs-config",
							MountPath: "/config",
						},
						{
							Name:      DataVolName,
							MountPath: "/export",
						},
					},
					SecurityContext: &corev1.SecurityContext{
						Capabilities: &corev1.Capabilities{
							Add: []corev1.Capability{
								"SYS_ADMIN",
								"DAC_READ_SEARCH",
							},
						},
					},
					ReadinessProbe: &corev1.Probe{
						Handler: corev1.Handler{
							HTTPGet: &corev1.HTTPGetAction{
								Port: intstr.FromInt(httpPort),
								Path: HealthEndpointPath,
							},
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "nfs-config",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: d.nfsServer.Name,
							},
						},
					},
				},
			},
		},
	}
}
