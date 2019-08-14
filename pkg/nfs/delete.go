package nfs

import (
	"github.com/storageos/cluster-operator/pkg/util"
)

// Delete deletes all the storageos resources.
// This explicit delete is implemented instead of depending on the garbage
// collector because sometimes the garbage collector deletes the resources
// with owner reference as a CRD without the parent being deleted. This happens
// especially when a cluster reboots. Althrough the operator re-creates the
// resources, we want to avoid this behavior by implementing an explcit delete.
func (d *Deployment) Delete() error {
	if err := util.DeleteStatefulSet(d.client, d.nfsServer.Name, d.nfsServer.Namespace); err != nil {
		return err
	}
	if err := util.DeleteConfigMap(d.client, d.nfsServer.Name, d.nfsServer.Namespace); err != nil {
		return err
	}
	if err := util.DeleteService(d.client, d.nfsServer.Name, d.nfsServer.Namespace); err != nil {
		return err
	}
	if err := util.DeleteClusterRoleBinding(d.client, d.getClusterRoleBindingName()); err != nil {
		return err
	}
	if err := util.DeleteServiceAccount(d.client, d.getServiceAccountName(), d.nfsServer.Namespace); err != nil {
		return err
	}

	// Maybe delete PVC as well.

	return nil
}
