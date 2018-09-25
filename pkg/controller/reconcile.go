package controller

import (
	"context"
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	api "github.com/storageos/storageoscluster-operator/pkg/apis/cluster/v1alpha1"
	"github.com/storageos/storageoscluster-operator/pkg/storageos"
)

// OperatorClient is an adapter that implements client.Client interface for operator-SDK.
type OperatorClient struct{}

// Create implements client.Client.
func (oc OperatorClient) Create(ctx context.Context, obj runtime.Object) error {
	return sdk.Create(obj)
}

// Update implements client.Client.
func (oc OperatorClient) Update(ctx context.Context, obj runtime.Object) error {
	return sdk.Update(obj)
}

// Delete implements client.Client.
func (oc OperatorClient) Delete(ctx context.Context, obj runtime.Object) error {
	return sdk.Delete(obj)
}

// Get implements client.Client.
func (oc OperatorClient) Get(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
	// operator-SDK refers namespace and name from the runtime object. Ignore
	// NamespacedName. sdk.GetOption is not passed at the moment.
	return sdk.Get(obj)
}

// List implements client.Client.
func (oc OperatorClient) List(ctx context.Context, opts *client.ListOptions, obj runtime.Object) error {
	// operator-SDK requires namespace to be passed separately. sdk.ListOption
	// is not passed at the moment.
	return sdk.List(opts.Namespace, obj)
}

// Status implements client.Client.
func (oc OperatorClient) Status() client.StatusWriter {
	return nil
}

// ClusterController is the StorageOS cluster controller.
type ClusterController struct {
	client client.Client
}

// NewClusterController creates and returns a new ClusterController, given a client.
func NewClusterController(c client.Client) *ClusterController {
	return &ClusterController{client: c}
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
