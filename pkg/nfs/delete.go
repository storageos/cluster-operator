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
	if err := d.k8sResourceManager.Service(d.nfsServer.Name, d.nfsServer.Namespace, nil, nil, nil).Delete(); err != nil {
		return err
	}
	if err := d.k8sResourceManager.Service(d.getMetricsServiceName(), d.nfsServer.Namespace, nil, nil, nil).Delete(); err != nil {
		return err
	}
	if err := d.k8sResourceManager.ClusterRoleBinding(d.getClusterRoleBindingName(), nil, nil).Delete(); err != nil {
		return err
	}
	if err := d.k8sResourceManager.ServiceAccount(d.getServiceAccountName(), d.nfsServer.Namespace).Delete(); err != nil {
		return err
	}

	// Delete PVC if it was not specified and dynamically created for NFS
	// Server.
	// NOTE: Reclaim policy is not respected here because NFS Server need not
	// have its own reclaim policy options. The StorageClass reclaim policy
	// must be used to set volume reclaim policy. NFS Server spec reclaim policy
	// will be removed in StorageOS cluster-operator v2 APIs.
	if d.nfsServer.Spec.PersistentVolumeClaim.ClaimName == "" {
		if err := d.k8sResourceManager.PersistentVolumeClaim(d.nfsServer.Name, d.nfsServer.Namespace, nil).Delete(); err != nil {
			return err
		}
	}

	return nil
}
