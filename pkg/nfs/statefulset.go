package nfs

import (
	"github.com/storageos/cluster-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// PVCNamePrefix is the prefix of the PVC names used by the NFS StatefulSet.
	// The PVC names are of the format <prefix>-<statefulset-pod-name>.
	// If the NFS server statefulset name is "example-nfsserver" and the pod is
	// named "example-nfsserver-0", the PVC will be named
	// "nfs-data-example-nfsserver-0"
	PVCNamePrefix = "nfs-data"
)

func (d *Deployment) createStatefulSet(size *resource.Quantity, nfsPort int, metricsPort int) error {

	replicas := int32(1)

	// TODO: Check if the PVC already exists before attempting to create one.

	spec := &appsv1.StatefulSetSpec{
		ServiceName: d.nfsServer.Name,
		Replicas:    &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: d.labelsForStatefulSet(d.nfsServer.Name, d.nfsServer.Labels),
		},
		Template:             d.createPodTemplateSpec(nfsPort, metricsPort, d.nfsServer.Labels),
		VolumeClaimTemplates: d.createVolumeClaimTemplateSpecs(size, d.nfsServer.Labels),
	}

	// TODO: Add node affinity support for NFS server pods.
	util.AddTolerations(&spec.Template.Spec, d.nfsServer.Spec.Tolerations)

	return d.k8sResourceManager.StatefulSet(d.nfsServer.Name, d.nfsServer.Namespace, spec).Create()
}

func (d *Deployment) createVolumeClaimTemplateSpecs(size *resource.Quantity, labels map[string]string) []corev1.PersistentVolumeClaim {
	scName := d.nfsServer.Spec.StorageClassName

	claim := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PVCNamePrefix,
			Namespace: d.nfsServer.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &scName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{},
			},
		},
	}

	if size != nil {
		claim.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceName(corev1.ResourceStorage): *size,
		}
	}

	return []corev1.PersistentVolumeClaim{claim}
}

func (d *Deployment) createPodTemplateSpec(nfsPort int, metricsPort int, labels map[string]string) corev1.PodTemplateSpec {

	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: d.labelsForStatefulSet(d.nfsServer.Name, labels),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: d.getServiceAccountName(),
			Containers: []corev1.Container{
				{
					ImagePullPolicy: "IfNotPresent",
					Name:            "nfsd",
					Image:           d.nfsServer.Spec.GetContainerImage(),
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
							Name:          "metrics-port",
							ContainerPort: int32(metricsPort),
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "nfs-config",
							MountPath: "/config",
						},
						{
							Name:      "nfs-data",
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
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "nfs-config",
					VolumeSource: v1.VolumeSource{
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
