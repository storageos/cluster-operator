package storageos

import (
	"strings"

	"github.com/storageos/cluster-operator/pkg/util"
)

// Delete deletes all the storageos resources.
// This explicit delete is implemented instead of depending on the garbage
// collector because sometimes the garbage collector deletes the resources
// with owner reference as a CRD without the parent being deleted. This happens
// especially when a cluster reboots. Althrough the operator re-creates the
// resources, we want to avoid this behavior by implementing an explcit delete.
func (s *Deployment) Delete() error {
	namespace := s.stos.Spec.GetResourceNS()

	if err := util.DeleteStorageClass(s.client, s.stos.Spec.GetStorageClassName()); err != nil {
		return err
	}

	if err := util.DeleteService(s.client, s.stos.Spec.GetServiceName(), namespace); err != nil {
		return err
	}

	if err := util.DeleteDaemonSet(s.client, daemonsetName, namespace); err != nil {
		return err
	}

	if err := util.DeleteSecret(s.client, initSecretName, namespace); err != nil {
		return err
	}

	if err := util.DeleteRoleBinding(s.client, KeyManagementBindingName, namespace); err != nil {
		return err
	}

	if err := util.DeleteRole(s.client, KeyManagementRoleName, namespace); err != nil {
		return err
	}

	if err := util.DeleteServiceAccount(s.client, DaemonsetSA, namespace); err != nil {
		return err
	}

	if s.stos.Spec.CSI.Enable {
		if err := s.deleteCSIHelper(); err != nil {
			return err
		}

		if err := util.DeleteClusterRoleBinding(s.client, CSIAttacherClusterBindingName); err != nil {
			return err
		}

		if err := util.DeleteClusterRoleBinding(s.client, CSIProvisionerClusterBindingName); err != nil {
			return err
		}

		if err := util.DeleteClusterRole(s.client, CSIAttacherClusterRoleName); err != nil {
			return err
		}

		if err := util.DeleteClusterRole(s.client, CSIProvisionerClusterRoleName); err != nil {
			return err
		}

		if err := util.DeleteServiceAccount(s.client, s.getCSIHelperServiceAccountName(), namespace); err != nil {
			return err
		}

		if err := util.DeleteClusterRoleBinding(s.client, CSIK8SDriverRegistrarClusterBindingName); err != nil {
			return err
		}

		if err := util.DeleteClusterRoleBinding(s.client, CSIDriverRegistrarClusterBindingName); err != nil {
			return err
		}

		if err := util.DeleteClusterRole(s.client, CSIDriverRegistrarClusterRoleName); err != nil {
			return err
		}

		if err := s.deleteCSISecrets(); err != nil {
			return err
		}
	}

	if !s.stos.Spec.DisableScheduler {
		if err := s.deleteSchedulerExtender(); err != nil {
			return err
		}
	}

	// Delete cluster role for openshift security context constraints.
	if strings.Contains(s.stos.Spec.K8sDistro, K8SDistroOpenShift) {
		if err := util.DeleteClusterRoleBinding(s.client, OpenShiftSCCClusterBindingName); err != nil {
			return err
		}

		if err := util.DeleteClusterRole(s.client, OpenShiftSCCClusterRoleName); err != nil {
			return err
		}
	}

	// Delete role for Pod Fencing.
	if !s.stos.Spec.DisableFencing {
		if err := util.DeleteClusterRoleBinding(s.client, FencingClusterBindingName); err != nil {
			return err
		}

		if err := util.DeleteClusterRole(s.client, FencingClusterRoleName); err != nil {
			return err
		}
	}

	// NOTE: Do not delete the namespace. Namespace can have some resources
	// created by the control plane. They must not be deleted.

	return nil
}
