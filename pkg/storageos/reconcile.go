package storageos

import (
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"

	api "github.com/storageos/storageoscluster-operator/pkg/apis/cluster/v1alpha1"
)

// Reconcile ensures that the state specified in the Spec of the object matches
// the state of the system.
func Reconcile(m *api.StorageOSCluster, recorder record.EventRecorder) error {
	// Get a new list of nodes and update the join token with new nodes.
	nodeList := nodeList()
	if err := sdk.List(m.Spec.GetResourceNS(), nodeList); err != nil {
		return fmt.Errorf("failed to list nodes: %v", err)
	}

	nodeIPs := getNodeIPs(nodeList.Items)
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
		if err := deployStorageOS(m, recorder); err != nil {
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
		return sdk.Update(m)
	}

	return nil
}
