package stub

import (
	"context"

	"github.com/storageos/storageoscluster-operator/pkg/apis/cluster/v1alpha1"
	"github.com/storageos/storageoscluster-operator/pkg/controller"
	"k8s.io/client-go/tools/record"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// NewHandler returns a new event handler given a recorder and controller.
func NewHandler(eRec record.EventRecorder, c *controller.ClusterController) sdk.Handler {
	return &Handler{eventRecorder: eRec, controller: c}
}

// Handler contains the controller and event broadcast recorder.
type Handler struct {
	eventRecorder record.EventRecorder
	controller    *controller.ClusterController
}

// Handle calls the controller reconcile method based on the event.
func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.StorageOSCluster:

		// Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// All secondary resources must have the CR set as their OwnerReference for this to be the case
		if event.Deleted {
			return nil
		}

		return h.controller.Reconcile(o, h.eventRecorder)
	}

	return nil
}
