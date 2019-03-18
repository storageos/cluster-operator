package storageos

// Delete deletes all the storageos resources.
// This explicit delete is implemented instead of depending on the garbage
// collector because sometimes the garbage collector deletes the resources
// with owner reference as a CRD without the parent being deleted. This happens
// especially when a cluster reboots. Althrough the operator re-creates the
// resources, we want to avoid this behavior by implementing an explcit delete.
func (s *Deployment) Delete() error {

	if err := s.deleteStorageClass("fast"); err != nil {
		return err
	}

	if err := s.deleteService(s.stos.Spec.GetServiceName()); err != nil {
		return err
	}

	if err := s.deleteDaemonSet(daemonsetName); err != nil {
		return err
	}

	if err := s.deleteSecret(initSecretName); err != nil {
		return err
	}

	if err := s.deleteRoleBinding(keyManagementBindingName); err != nil {
		return err
	}

	if err := s.deleteRole(keyManagementRoleName); err != nil {
		return err
	}

	if err := s.deleteServiceAccount("storageos-daemonset-sa"); err != nil {
		return err
	}

	if s.stos.Spec.CSI.Enable {
		if err := s.deleteStatefulSet(statefulsetName); err != nil {
			return err
		}

		if err := s.deleteClusterRoleBinding("csi-attacher-binding"); err != nil {
			return err
		}

		if err := s.deleteClusterRoleBinding("csi-provisioner-binding"); err != nil {
			return err
		}

		if err := s.deleteClusterRole("csi-attacher-role"); err != nil {
			return err
		}

		if err := s.deleteClusterRole("csi-provisioner-role"); err != nil {
			return err
		}

		if err := s.deleteServiceAccount("storageos-statefulset-sa"); err != nil {
			return err
		}

		if err := s.deleteClusterRoleBinding("k8s-driver-registrar-binding"); err != nil {
			return err
		}

		if err := s.deleteClusterRoleBinding("driver-registrar-binding"); err != nil {
			return err
		}

		if err := s.deleteClusterRole("driver-registrar-role"); err != nil {
			return err
		}

		if err := s.deleteCSISecrets(); err != nil {
			return err
		}
	}

	// NOTE: Do not delete the namespace. Namespace can have some resources
	// created by the control plane. They must not be deleted.

	return nil
}
