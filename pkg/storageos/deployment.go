package storageos

import (
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/cluster-operator/pkg/util/k8s"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Deployment stores all the resource configuration and performs
// resource creation and update.
type Deployment struct {
	client             client.Client
	stos               *storageosv1.StorageOSCluster
	recorder           record.EventRecorder
	k8sVersion         string
	scheme             *runtime.Scheme
	update             bool
	k8sResourceManager *k8s.ResourceManager
}

// NewDeployment creates a new Deployment given a k8c client, storageos manifest
// and an event broadcast recorder.
func NewDeployment(
	client client.Client,
	stos *storageosv1.StorageOSCluster,
	labels map[string]string,
	recorder record.EventRecorder,
	scheme *runtime.Scheme,
	version string,
	update bool) *Deployment {
	return &Deployment{
		client:             client,
		stos:               stos,
		recorder:           recorder,
		k8sVersion:         version,
		scheme:             scheme,
		update:             update,
		k8sResourceManager: k8s.NewResourceManager(client).SetLabels(labels),
	}
}
