package storageos

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/cluster-operator/pkg/util/k8s"
)

// Deployment stores all the resource configuration and performs
// resource creation and update.
type Deployment struct {
	client             client.Client
	discoveryClient    discovery.DiscoveryInterface
	stos               *storageosv1.StorageOSCluster
	recorder           record.EventRecorder
	k8sVersion         string
	scheme             *runtime.Scheme
	update             bool
	k8sResourceManager *k8s.ResourceManager
	nodev2             bool
}

// NewDeployment creates a new Deployment given a k8c client, storageos manifest
// and an event broadcast recorder.
func NewDeployment(
	client client.Client,
	discoveryClient discovery.DiscoveryInterface,
	stos *storageosv1.StorageOSCluster,
	labels map[string]string,
	recorder record.EventRecorder,
	scheme *runtime.Scheme,
	version string,
	update bool) *Deployment {
	return &Deployment{
		client:             client,
		discoveryClient:    discoveryClient,
		stos:               stos,
		recorder:           recorder,
		k8sVersion:         version,
		scheme:             scheme,
		update:             update,
		k8sResourceManager: k8s.NewResourceManager(client).SetLabels(labels),
		nodev2:             isV2image(stos.Spec.Images.NodeContainer),
	}
}

// isV2image returns true if the image tag starts with "2." or contains "c2".
func isV2image(image string) bool {

	parts := strings.Split(image, ":")
	if len(parts) < 2 {
		return false
	}

	if strings.HasPrefix(parts[len(parts)-1], "2.") {
		return true
	}

	// Temporary dev tag check.
	// TODO: remove once we have proper tags.
	return strings.Contains(parts[len(parts)-1], "c2")
}
