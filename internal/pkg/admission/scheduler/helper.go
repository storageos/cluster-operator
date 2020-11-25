package scheduler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	// pvcStorageClassKey is the annotation used to refer to the StorageClass when
	// the PVC storageClassName wasn't used.  This is now deprecated but should
	// still be checked as k8s still supports it.
	pvcStorageClassKey = "volume.beta.kubernetes.io/storage-class"

	// defaultStorageClassKey is the annotation used to denote whether a
	// StorageClass is the cluster default.
	defaultStorageClassKey = "storageclass.kubernetes.io/is-default-class"
)

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

	// Get the StorageClass that provisioned the volume.
	sc, err := p.getPVCStorageClass(pvc)
	if err != nil {
		return false, err
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

// getPVCStorageClass returns the StorageClass of the PVC.  If no StorageClass
// was specified, returns the cluster default if set.
func (p *PodSchedulerSetter) getPVCStorageClass(pvc *corev1.PersistentVolumeClaim) (*storagev1.StorageClass, error) {
	scName := getPVCStorageClassName(pvc)
	if scName == "" {
		return p.getDefaultStorageClass()
	}
	sc := &storagev1.StorageClass{}
	scNSName := types.NamespacedName{
		Name: scName,
	}
	if err := p.client.Get(context.Background(), scNSName, sc); err != nil {
		return nil, fmt.Errorf("failed to get StorageClass: %v", err)
	}
	return sc, nil
}

// getDefaultStorageClass returns the default StorageClass, if any.
func (p *PodSchedulerSetter) getDefaultStorageClass() (*storagev1.StorageClass, error) {
	scList := &storagev1.StorageClassList{}
	if err := p.client.List(context.Background(), scList, &client.ListOptions{}); err != nil {
		return nil, fmt.Errorf("failed to get StorageClasses: %v", err)
	}
	for _, sc := range scList.Items {
		if val, ok := sc.Annotations[defaultStorageClassKey]; ok && val == "true" {
			return &sc, nil
		}
	}
	return nil, fmt.Errorf("default StorageClass not found")
}

// getPVCStorageClassName returns the PVC provisioner name.
func getPVCStorageClassName(pvc *corev1.PersistentVolumeClaim) string {
	// The beta annotation should still be supported since even latest versions
	// of Kubernetes still allow it.
	if pvc.Spec.StorageClassName != nil && len(*pvc.Spec.StorageClassName) > 0 {
		return *pvc.Spec.StorageClassName
	}
	if val, ok := pvc.Annotations[pvcStorageClassKey]; ok {
		return val
	}
	return ""
}
