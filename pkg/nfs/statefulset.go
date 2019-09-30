package nfs

import (
	"github.com/storageos/cluster-operator/pkg/util"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// PVCNamePrefix is the prefix of the PVC names used by the NFS StatefulSet.
	// The PVC names are of the format <prefix>-<statefulset-pod-name>.
	// If the NFS server statefulset name is "example-nfsserver" and the pod is
	// named "example-nfsserver-0", the PVC will be named
	// "nfs-data-example-nfsserver-0"
	PVCNamePrefix = "nfs-data"
)

func (d *Deployment) createStatefulSet(size *resource.Quantity, nfsPort int, httpPort int) error {

	replicas := int32(1)

	spec := &appsv1.StatefulSetSpec{
		ServiceName: d.nfsServer.Name,
		Replicas:    &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: d.labelsForStatefulSet(d.nfsServer.Name, d.nfsServer.Labels),
		},
		Template: d.createPodTemplateSpec(nfsPort, httpPort),
	}

	// If no existing PVC is specified in the spec, create volume claim template
	// for a new PVC.
	if d.nfsServer.Spec.PersistentVolumeClaim.ClaimName == "" {
		spec.VolumeClaimTemplates = d.createVolumeClaimTemplateSpecs(size)
	} else {
		// If a PVC is provided in the NFSServer CR, add a reference to the PVC
		// in the volumes list.
		pvc := corev1.Volume{
			Name: PVCNamePrefix,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &d.nfsServer.Spec.PersistentVolumeClaim,
			},
		}
		spec.Template.Spec.Volumes = append(spec.Template.Spec.Volumes, pvc)
	}

	// TODO: Add node affinity support for NFS server pods.
	util.AddTolerations(&spec.Template.Spec, d.nfsServer.Spec.Tolerations)

	return d.k8sResourceManager.StatefulSet(d.nfsServer.Name, d.nfsServer.Namespace, spec).Create()
}

func (d *Deployment) createVolumeClaimTemplateSpecs(size *resource.Quantity) []corev1.PersistentVolumeClaim {
	scName := d.nfsServer.Spec.GetStorageClassName(d.cluster.Spec.GetStorageClassName())

	claim := corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      PVCNamePrefix,
			Namespace: d.nfsServer.Namespace,
			Labels:    d.nfsServer.Labels,
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

func (d *Deployment) createPodTemplateSpec(nfsPort int, httpPort int) corev1.PodTemplateSpec {
	return corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: d.labelsForStatefulSet(d.nfsServer.Name, d.nfsServer.Labels),
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
							Name:      PVCNamePrefix,
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
