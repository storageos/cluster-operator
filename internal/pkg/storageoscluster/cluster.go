package storageoscluster

import (
	"context"
	"errors"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

// ErrNoCluster is the error when there's no running StorageOS cluster found.
var ErrNoCluster = errors.New("no storageos cluster found")

// GetCurrentStorageOSCluster returns the currently running StorageOS cluster.
func GetCurrentStorageOSCluster(kclient client.Client) (*storageosv1.StorageOSCluster, error) {
	var currentCluster *storageosv1.StorageOSCluster

	// Get a list of all the StorageOS clusters.
	clusterList := &storageosv1.StorageOSClusterList{}
	listOpts := []client.ListOption{}
	if err := kclient.List(context.Background(), clusterList, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list storageos clusters: %v", err)
	}

	// If there's only one cluster, return it as the current cluster.
	if len(clusterList.Items) == 1 {
		currentCluster = &clusterList.Items[0]
	}

	// If there are multiple clusters, consider the status of the cluster.
	for _, cluster := range clusterList.Items {
		cluster := cluster
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
