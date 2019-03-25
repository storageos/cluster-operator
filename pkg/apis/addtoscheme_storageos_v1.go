package apis

import (
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, storageosv1.SchemeBuilder.AddToScheme)
}
