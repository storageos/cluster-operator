package storageos

import (
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	kdiscovery "k8s.io/client-go/discovery"

	"github.com/storageos/cluster-operator/internal/pkg/discovery"
	k8sresource "github.com/storageos/cluster-operator/pkg/util/k8s/resource"
)

// createCSIDriver creates a StorageOS CSIDriver resource with the required
// attributes.
func (s *Deployment) createCSIDriver() error {
	attachRequired := true
	podInfoRequired := true

	spec := &storagev1beta1.CSIDriverSpec{
		AttachRequired: &attachRequired,
		PodInfoOnMount: &podInfoRequired,
	}

	return k8sresource.NewCSIDriver(s.client, CSIProvisionerName, nil, spec).Create()
}

// deleteCSIDriver deletes the StorageOS CSIDriver resource.
func (s Deployment) deleteCSIDriver() error {
	return s.k8sResourceManager.CSIDriver(CSIProvisionerName, nil, nil).Delete()
}

// HasCSIDriverKind checks if CSIDriver built-in resource is supported in the
// k8s cluster.
func HasCSIDriverKind(dc kdiscovery.DiscoveryInterface) (bool, error) {
	return discovery.HasResource(dc, k8sresource.APIstoragev1beta1, k8sresource.CSIDriverKind)
}
