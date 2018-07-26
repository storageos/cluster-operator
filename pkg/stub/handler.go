package stub

import (
	"context"

	"github.com/storageos/storageos-operator/pkg/apis/node/v1alpha1"
	"github.com/storageos/storageos-operator/pkg/storageos"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

func NewHandler() sdk.Handler {
	return &Handler{}
}

type Handler struct {
	// Fill me
}

func (h *Handler) Handle(ctx context.Context, event sdk.Event) error {
	switch o := event.Object.(type) {
	case *v1alpha1.StorageOS:

		// Ignore the delete event since the garbage collector will clean up all secondary resources for the CR
		// All secondary resources must have the CR set as their OwnerReference for this to be the case
		if event.Deleted {
			return nil
		}

		return storageos.Reconcile(o)
	}

	return nil
}
