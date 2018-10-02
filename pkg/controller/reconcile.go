package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	clusterv1alpha1 "github.com/storageos/storageoscluster-operator/pkg/apis/cluster/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/storageos/storageoscluster-operator/pkg/apis/cluster/v1alpha1"
	"github.com/storageos/storageoscluster-operator/pkg/storageos"
)

// ClusterController is the StorageOS cluster controller.
type ClusterController struct {
	client         client.Client
	currentCluster *clusterv1alpha1.StorageOSCluster
}

// NewClusterController creates and returns a new ClusterController, given a client.
func NewClusterController(c client.Client) *ClusterController {
	return &ClusterController{client: c}
}

// SetCurrentClusterIfNone checks if there's any existing current cluster and
// sets a new current cluster if it wasn't set before.
func (c *ClusterController) SetCurrentClusterIfNone(cluster *clusterv1alpha1.StorageOSCluster) {
	if c.currentCluster == nil {
		c.SetCurrentCluster(cluster)
	}
}

// SetCurrentCluster sets the currently active cluster in the controller.
func (c *ClusterController) SetCurrentCluster(cluster *clusterv1alpha1.StorageOSCluster) {
	c.currentCluster = cluster
}

// IsCurrentCluster compares a given cluster with the current cluster to check
// if they are the same.
func (c *ClusterController) IsCurrentCluster(cluster *clusterv1alpha1.StorageOSCluster) bool {
	if cluster == nil {
		return false
	}

	if (c.currentCluster.GetName() == cluster.GetName()) && (c.currentCluster.GetNamespace() == cluster.GetNamespace()) {
		return true
	}
	return false
}

// ResetCurrentCluster resets the current cluster of the controller.
func (c *ClusterController) ResetCurrentCluster() {
	cleanup(c.client)
	c.currentCluster = nil
}

// Reconcile ensures that the state specified in the Spec of the object matches
// the state of the system.
func (c *ClusterController) Reconcile(m *api.StorageOSCluster, recorder record.EventRecorder) error {
	// Get a new list of nodes and update the join token with new nodes.
	nodeList := storageos.NodeList()
	if err := sdk.List(m.Spec.GetResourceNS(), nodeList); err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	nodeIPs := storageos.GetNodeIPs(nodeList.Items)
	join := strings.Join(nodeIPs, ",")

	if m.Spec.Join != join {
		m.Spec.Join = join
		// Update Nodes as well, because updating StorageOS with null Nodes
		// results in invalid config.
		m.Status.Nodes = nodeIPs
		if err := sdk.Update(m); err != nil {
			return err
		}

	}

	// Update the spec values. This ensures that the default values are applied
	// when fields are not set in the spec.
	m.Spec.ResourceNS = m.Spec.GetResourceNS()
	m.Spec.Images.NodeContainer = m.Spec.GetNodeContainerImage()
	m.Spec.Images.InitContainer = m.Spec.GetInitContainerImage()

	if m.Spec.CSI.Enable {
		m.Spec.Images.CSIDriverRegistrarContainer = m.Spec.GetCSIDriverRegistrarImage()
		m.Spec.Images.CSIExternalProvisionerContainer = m.Spec.GetCSIExternalProvisionerImage()
		m.Spec.Images.CSIExternalAttacherContainer = m.Spec.GetCSIExternalAttacherImage()
	}

	if m.Spec.Ingress.Enable {
		m.Spec.Ingress.Hostname = m.Spec.GetIngressHostname()
	}

	m.Spec.Service.Name = m.Spec.GetServiceName()
	m.Spec.Service.Type = m.Spec.GetServiceType()
	m.Spec.Service.ExternalPort = m.Spec.GetServiceExternalPort()
	m.Spec.Service.InternalPort = m.Spec.GetServiceInternalPort()

	// Finalizers are set when an object should be deleted. Apply deploy only
	// when finalizers is empty.
	if len(m.GetFinalizers()) == 0 {
		stosDeployment := storageos.NewDeployment(c.client, m, recorder)
		if err := stosDeployment.Deploy(); err != nil {
			// Ignore "Operation cannot be fulfilled" error. It happens when the
			// actual state of object is different from what is known to the operator.
			// Operator would resync and retry the failed operation on its own.
			if !strings.HasPrefix(err.Error(), "Operation cannot be fulfilled") {
				recorder.Event(m, v1.EventTypeWarning, "FailedCreation", err.Error())
			}
			return err
		}
	} else {
		recorder.Event(m, v1.EventTypeNormal, "Terminating", "StorageOS object deleted")
		c.ResetCurrentCluster()
		// Reset finalizers and let k8s delete the object.
		// When finalizers are set on an object, metadata.deletionTimestamp is
		// also set. deletionTimestamp helps the garbage collector identify
		// when to delete an object. k8s deletes the object only once the
		// list of finalizers is empty.
		m.SetFinalizers([]string{})
		// return sdk.Update(m)
		return c.client.Update(context.Background(), m)
	}

	return nil
}
