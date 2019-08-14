package nfs

import (
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	stosClientset "github.com/storageos/cluster-operator/pkg/client/clientset/versioned"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Deployment struct {
	client     client.Client
	nfsServer  *storageosv1.NFSServer
	recorder   record.EventRecorder
	scheme     *runtime.Scheme
	stosClient stosClientset.Interface
	cluster    *storageosv1.StorageOSCluster
}

func NewDeployment(client client.Client, stosClient stosClientset.Interface, nfsServer *storageosv1.NFSServer, recorder record.EventRecorder, scheme *runtime.Scheme) *Deployment {
	return &Deployment{
		client:     client,
		nfsServer:  nfsServer,
		recorder:   recorder,
		scheme:     scheme,
		stosClient: stosClient,
	}
}
