package controller

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	clusterv1alpha1 "github.com/storageos/cluster-operator/pkg/apis/cluster/v1alpha1"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/storageos/cluster-operator/pkg/apis/cluster/v1alpha1"
	"github.com/storageos/cluster-operator/pkg/storageos"
)

// ClusterController is the StorageOS cluster controller.
type ClusterController struct {
	client         client.Client
	currentCluster *clusterv1alpha1.StorageOSCluster
	k8sVersion     string
}

// NewClusterController creates and returns a new ClusterController, given a client.
func NewClusterController(c client.Client, version string) *ClusterController {
	return &ClusterController{client: c, k8sVersion: version}
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
	if c.currentCluster.Spec.CleanupAtDelete {
		if err := cleanup(c.client, c.currentCluster); err != nil {
			// This error is just logged and not returned. Failing to cleanup
			// need not fail cluster reset.
			log.Println(err)
		}
	}
	c.currentCluster = nil
}

// Reconcile ensures that the state specified in the Spec of the object matches
// the state of the system.
func (c *ClusterController) Reconcile(m *api.StorageOSCluster, recorder record.EventRecorder) error {
	// Do not reconcile, the operator is paused for the cluster.
	if m.Spec.Pause {
		return nil
	}

	join, err := c.generateJoinToken(m)
	if err != nil {
		return err
	}

	if m.Spec.Join != join {
		m.Spec.Join = join
		// Update Nodes as well, because updating StorageOS with null Nodes
		// results in invalid config.
		m.Status.Nodes = strings.Split(join, ",")
		if err := sdk.Update(m); err != nil {
			return err
		}
	}

	// Update the spec values. This ensures that the default values are applied
	// when fields are not set in the spec.
	m.Spec.ResourceNS = m.Spec.GetResourceNS()
	m.Spec.Images.NodeContainer = m.Spec.GetNodeContainerImage()
	m.Spec.Images.InitContainer = m.Spec.GetInitContainerImage()
	m.Spec.Images.CleanupContainer = m.Spec.GetCleanupContainerImage()

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
		stosDeployment := storageos.NewDeployment(c.client, m, recorder, c.k8sVersion)
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

// generateJoinToken performs node selection based on NodeSelectorTerms if
// specified, and forms a join token by combining the node IPs.
func (c *ClusterController) generateJoinToken(m *api.StorageOSCluster) (string, error) {
	// Get a new list of all the nodes.
	nodeList := storageos.NodeList()
	if err := sdk.List(m.Spec.GetResourceNS(), nodeList); err != nil {
		return "", fmt.Errorf("failed to list nodes: %v", err)
	}

	selectedNodes := []v1.Node{}

	// Filter the node list when a node selector is applied.
	if len(m.Spec.NodeSelectorTerms) > 0 {
		for _, node := range nodeList.Items {
			// Skip a node with any taints. StorageOS pods don't support any
			// toleration.
			if len(node.Spec.Taints) > 0 {
				continue
			}
			for _, term := range m.Spec.NodeSelectorTerms {
				for _, exp := range term.MatchExpressions {
					var ex selection.Operator

					// Convert the node selector operator into requirement
					// selection operator.
					switch exp.Operator {
					case v1.NodeSelectorOpIn:
						ex = selection.Equals
					case v1.NodeSelectorOpNotIn:
						ex = selection.NotEquals
					}

					// Create a new Requirement to perform label matching.
					req, err := labels.NewRequirement(exp.Key, ex, exp.Values)
					if err != nil {
						return "", fmt.Errorf("failed to create requirement: %v", err)
					}

					if req.Matches(labels.Set(node.GetLabels())) {
						selectedNodes = append(selectedNodes, node)
					}
				}
			}
		}
	} else {
		selectedNodes = nodeList.Items
	}

	nodeIPs := storageos.GetNodeIPs(selectedNodes)
	return strings.Join(nodeIPs, ","), nil
}
