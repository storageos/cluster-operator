package storageos

import (
	api "github.com/storageos/storageos-operator/pkg/apis/node/v1alpha1"
)

func Reconcile(m *api.StorageOS) error {
	if err := deployStorageOS(m); err != nil {
		return err
	}

	return nil
}
