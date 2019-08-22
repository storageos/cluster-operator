package nfs

// Delete deletes all the storageos resources.
// This explicit delete is implemented instead of depending on the garbage
// collector because sometimes the garbage collector deletes the resources
// with owner reference as a CRD without the parent being deleted. This happens
// especially when a cluster reboots. Althrough the operator re-creates the
// resources, we want to avoid this behavior by implementing an explcit delete.
func (d *Deployment) Delete() error {
	if err := d.k8sResourceManager.StatefulSet(d.nfsServer.Name, d.nfsServer.Namespace, nil).Delete(); err != nil {
		return err
	}
	if err := d.k8sResourceManager.ConfigMap(d.nfsServer.Name, d.nfsServer.Namespace, nil).Delete(); err != nil {
		return err
	}
	if err := d.k8sResourceManager.Service(d.nfsServer.Name, d.nfsServer.Namespace, nil, nil).Delete(); err != nil {
		return err
	}
	if err := d.k8sResourceManager.ClusterRoleBinding(d.getClusterRoleBindingName(), nil, nil).Delete(); err != nil {
		return err
	}
	if err := d.k8sResourceManager.ServiceAccount(d.getServiceAccountName(), d.nfsServer.Namespace).Delete(); err != nil {
		return err
	}

	// Maybe delete PVC as well.

	return nil
}
