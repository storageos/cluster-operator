package nfs

import (
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/cluster-operator/pkg/util/k8s"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Deployment manages the NFS server deployment.
type Deployment struct {
	client             client.Client
	kConfig            *rest.Config
	nfsServer          *storageosv1.NFSServer
	recorder           record.EventRecorder
	scheme             *runtime.Scheme
	cluster            *storageosv1.StorageOSCluster
	k8sResourceManager *k8s.ResourceManager
}

// NewDeployment returns an initialized Deployment.
func NewDeployment(
	client client.Client,
	kConfig *rest.Config,
	stosCluster *storageosv1.StorageOSCluster,
	nfsServer *storageosv1.NFSServer,
	labels map[string]string,
	recorder record.EventRecorder,
	scheme *runtime.Scheme) *Deployment {

	return &Deployment{
		client:             client,
		kConfig:            kConfig,
		nfsServer:          nfsServer,
		recorder:           recorder,
		scheme:             scheme,
		cluster:            stosCluster,
		k8sResourceManager: k8s.NewResourceManager(client).SetLabels(labels),
	}
}
