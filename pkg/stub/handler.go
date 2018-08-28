package stub

import (
	"context"

	"github.com/storageos/storageos-operator/pkg/apis/node/v1alpha1"
	"github.com/storageos/storageos-operator/pkg/storageos"
	"k8s.io/client-go/tools/record"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

func NewHandler(eRec record.EventRecorder) sdk.Handler {
	return &Handler{eventRecorder: eRec}
}

type Handler struct {
	eventRecorder record.EventRecorder
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.StorageOS:

		// Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// All secondary resources must have the CR set as their OwnerReference for this to be the case
		if event.Deleted {
			return nil
		}

		return storageos.Reconcile(o, h.eventRecorder)
	}

	return nil
}
