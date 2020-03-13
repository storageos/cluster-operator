package node

import (
	"k8s.io/apimachinery/pkg/types"

	storageosclient "github.com/storageos/cluster-operator/internal/pkg/client/storageos"
)

// StorageOSClient stores storageos client related information.
type StorageOSClient struct {
	// client is the StorageOS API client.
	client storageosclient.Client

	// clusterName is the name of the current cluster.
	clusterName string
	// clusterGeneration is the StorageOSCluster resource's generation. This
	// number can be used to validate if the existing api client is still valid
	// for a cluster name. Any change in the cluster could change the storageos
	// API service. A new client must be created for a new generation of the
	// cluster.
	clusterGeneration int64
	// clusterUID is the UID of the cluster the client belongs to.
	clusterUID types.UID
}
