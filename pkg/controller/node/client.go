package node

import (
	storageosapi "github.com/storageos/go-api"
	"k8s.io/apimachinery/pkg/types"
)

// StorageOSClient stores storageos client related information.
type StorageOSClient struct {
	*storageosapi.Client
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
