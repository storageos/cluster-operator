package storageos

import (
	"github.com/operator-framework/operator-sdk/pkg/sdk"

	api "github.com/storageos/storageos-operator/pkg/apis/node/v1alpha1"
)

func Reconcile(m *api.StorageOS) error {
	// Finalizers are set when an object should be deleted. Apply deploy only
	// when finalizers is empty.
	if len(m.GetFinalizers()) == 0 {
		if err := deployStorageOS(m); err != nil {
			return err
		}
	} else {
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
