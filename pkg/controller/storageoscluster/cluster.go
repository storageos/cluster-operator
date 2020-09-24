package storageoscluster

import (
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/pkg/util/k8s"
)

// StorageOSCluster stores the current cluster's information. It binds the
// cluster and the deployment together, ensuring deployment interacts with the
// right cluster resource.
type StorageOSCluster struct {
	cluster *storageosv1.StorageOSCluster
	// deployment implements storageoscluster.Deployment interface. This is
	// cached for a cluster to avoid recreating it without any change to the
	// cluster object. Every new cluster will create its unique deployment.
	deployment Deployment
}

// NewStorageOSCluster creates a new StorageOSCluster object.
func NewStorageOSCluster(cluster *storageosv1.StorageOSCluster) *StorageOSCluster {
	return &StorageOSCluster{cluster: cluster}
}

// SetDeployment creates a new StorageOS Deployment and sets it for the current
// StorageOSCluster.
func (c *StorageOSCluster) SetDeployment(r *ReconcileStorageOSCluster) {
	// updateIfExists is set to false because we don't update any existing
	// resources for now. May change in future.
	// TODO: Change this after resolving the conflict between two way
	// binding and upgrade.
	updateIfExists := false

	// Labels to be applied on all the k8s resources that are created for
	// StorageOS deployment. Inherit the labels from the CR.
	labels := c.cluster.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	// Add default labels.
	// TODO: This is legacy label. Remove this with care. Ensure it's not used
	// by any label selectors.
	labels["app"] = "storageos"

	// Add default resource app labels.  Component will be set to "cluster" by
	// default.
	labels = k8s.AddDefaultAppLabels(c.cluster.Name, labels)

	c.deployment = storageos.NewDeployment(r.client, r.discoveryClient, c.cluster, labels, r.recorder, r.scheme, r.k8sVersion, updateIfExists)
}

// IsCurrentCluster compares the cluster attributes to check if the given
// cluster is the same as the current cluster.
func (c *StorageOSCluster) IsCurrentCluster(cluster *storageosv1.StorageOSCluster) bool {
	if (c.cluster.GetName() == cluster.GetName()) &&
		(c.cluster.GetNamespace() == cluster.GetNamespace()) {
		return true
	}
	return false
}

// Deploy deploys the StorageOS cluster.
func (c *StorageOSCluster) Deploy(r *ReconcileStorageOSCluster) error {
	if c.deployment == nil {
		c.SetDeployment(r)
	}
	return c.deployment.Deploy()
}

// DeleteDeployment deletes the StorageOS Cluster deployment.
func (c *StorageOSCluster) DeleteDeployment(r *ReconcileStorageOSCluster) error {
	if c.deployment == nil {
		c.SetDeployment(r)
	}
	return c.deployment.Delete()
}
