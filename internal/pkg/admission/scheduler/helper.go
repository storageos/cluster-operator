package scheduler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
)

// pvcStorageClassKey is the annotation used to refer to the StorageClass when
// the PVC storageClassName wasn't used.  This is now deprecated but should
// still be checked as k8s still supports it.
const pvcStorageClassKey = "volume.beta.kubernetes.io/storage-class"

// IsManagedVolume inspects a given volume to find if it's managed by the given
// provisioners.
func (p *PodSchedulerSetter) IsManagedVolume(volume corev1.Volume, namespace string) (bool, error) {
	// Ensure that the volume has a claim.
	if volume.PersistentVolumeClaim == nil {
		return false, nil
	}

	// Get the PersistentVolumeClaim object.
	pvc := &corev1.PersistentVolumeClaim{}
	pvcNSName := types.NamespacedName{
		Name:      volume.PersistentVolumeClaim.ClaimName,
		Namespace: namespace,
	}
	if err := p.client.Get(context.Background(), pvcNSName, pvc); err != nil {
		return false, fmt.Errorf("failed to get PVC: %v", err)
	}

	// Get the StorageClass of the PVC.  The beta annotation should still be
	// supported since even latest versions of Kubernetes still allow it.
	var scName string
	if pvc.Spec.StorageClassName != nil && len(*pvc.Spec.StorageClassName) > 0 {
		scName = *pvc.Spec.StorageClassName
	} else {
		scName = pvc.Annotations[pvcStorageClassKey]
	}
	if scName == "" {
		return false, fmt.Errorf("could not get StorageClass name associated with PVC %q", pvc.Name)
	}
	sc := &storagev1.StorageClass{}
	scNSName := types.NamespacedName{
		Name: scName,
	}
	if err := p.client.Get(context.Background(), scNSName, sc); err != nil {
		return false, fmt.Errorf("failed to get StorageClass: %v", err)
	}

	// Check if the StorageClass provisioner matches with any of the provided
	// provisioners.
	for _, provisioner := range p.Provisioners {
		if sc.Provisioner == provisioner {
			// This is a managed volume.
			return true, nil
		}
	}

	return false, nil
}
