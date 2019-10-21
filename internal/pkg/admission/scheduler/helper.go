package scheduler

import (
	"context"
	"errors"
	"fmt"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrNoCluster is the error when there's no running StorageOS cluster found.
var ErrNoCluster = errors.New("no storageos cluster found")

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

	// Get the StorageClass of the PVC.
	scName := pvc.Spec.StorageClassName
	if scName == nil {
		return false, fmt.Errorf("could not get StorageClass name associated with PVC %q", pvc.Name)
	}
	sc := &storagev1.StorageClass{}
	scNSName := types.NamespacedName{
		Name: *scName,
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

// getCurrentStorageOSCluster returns the currently running StorageOS cluster.
// TODO: Move this to a separate package as a helper function.
func (p *PodSchedulerSetter) getCurrentStorageOSCluster() (*storageosv1.StorageOSCluster, error) {
	var currentCluster *storageosv1.StorageOSCluster

	// Get a list of all the StorageOS clusters.
	clusterList := &storageosv1.StorageOSClusterList{}
	if err := p.client.List(context.Background(), &client.ListOptions{}, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list storageos clusters: %v", err)
	}

	// If there's only one cluster, return it as the current cluster.
	if len(clusterList.Items) == 1 {
		currentCluster = &clusterList.Items[0]
	}

	// If there are multiple clusters, consider the status of the cluster.
	for _, cluster := range clusterList.Items {
		// Only one cluster can be in running phase at a time.
		if cluster.Status.Phase == storageosv1.ClusterPhaseRunning {
			currentCluster = &cluster
			break
		}
	}

	// If no current cluster found, fail.
	if currentCluster != nil {
		return currentCluster, nil
	}

	return currentCluster, ErrNoCluster
}
