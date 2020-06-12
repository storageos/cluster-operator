package nfs

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func (d *Deployment) createPVC(size *resource.Quantity) error {
	scName := d.nfsServer.Spec.GetStorageClassName(d.cluster.Spec.GetStorageClassName())

	spec := &corev1.PersistentVolumeClaimSpec{
		AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		StorageClassName: &scName,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: *size,
			},
		},
	}

	return d.k8sResourceManager.PersistentVolumeClaim(d.nfsServer.Name, d.nfsServer.Namespace, nil, spec).Create()
}
