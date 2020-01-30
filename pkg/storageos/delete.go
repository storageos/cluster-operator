package storageos

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

// Delete deletes all the storageos resources.
// This explicit delete is implemented instead of depending on the garbage
// collector because sometimes the garbage collector deletes the resources
// with owner reference as a CRD without the parent being deleted. This happens
// especially when a cluster reboots. Althrough the operator re-creates the
// resources, we want to avoid this behavior by implementing an explcit delete.
func (s *Deployment) Delete() error {
	namespace := s.stos.Spec.GetResourceNS()

	if err := s.k8sResourceManager.StorageClass(s.stos.Spec.GetStorageClassName(), nil, "", nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.Service(s.stos.Spec.GetServiceName(), namespace, nil, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.DaemonSet(daemonsetName, namespace, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.ConfigMap(configmapName, namespace, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.Secret(initSecretName, namespace, nil, corev1.SecretTypeOpaque, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.ClusterRoleBinding(InitClusterBindingName, nil, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.ClusterRole(InitClusterRoleName, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.RoleBinding(KeyManagementBindingName, namespace, nil, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.Role(KeyManagementRoleName, namespace, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.RoleBinding(NFSClusterBindingName, namespace, nil, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.Role(NFSClusterRoleName, namespace, nil, nil).Delete(); err != nil {
		return err
	}

	if err := s.k8sResourceManager.ServiceAccount(DaemonsetSA, namespace, nil).Delete(); err != nil {
		return err
	}

	if s.stos.Spec.CSI.Enable {
		// Delete CSIDriver if supported.
		supportsCSIDriver, err := HasCSIDriverKind(s.discoveryClient)
		if err != nil {
			return err
		}
		if supportsCSIDriver {
			if err := s.deleteCSIDriver(); err != nil {
				return err
			}
		}

		if err := s.deleteCSIHelper(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRoleBinding(CSIAttacherClusterBindingName, nil, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRoleBinding(CSIProvisionerClusterBindingName, nil, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRole(CSIAttacherClusterRoleName, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRole(CSIProvisionerClusterRoleName, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ServiceAccount(s.getCSIHelperServiceAccountName(), namespace, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRoleBinding(CSIK8SDriverRegistrarClusterBindingName, nil, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRoleBinding(CSIDriverRegistrarClusterBindingName, nil, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRole(CSIDriverRegistrarClusterRoleName, nil, nil).Delete(); err != nil {
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
		if err := s.k8sResourceManager.ClusterRoleBinding(OpenShiftSCCClusterBindingName, nil, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRole(OpenShiftSCCClusterRoleName, nil, nil).Delete(); err != nil {
			return err
		}
	}

	// Delete role for Pod Fencing.
	if !s.stos.Spec.DisableFencing {
		if err := s.k8sResourceManager.ClusterRoleBinding(FencingClusterBindingName, nil, nil, nil).Delete(); err != nil {
			return err
		}

		if err := s.k8sResourceManager.ClusterRole(FencingClusterRoleName, nil, nil).Delete(); err != nil {
			return err
		}
	}

	// NOTE: Do not delete the namespace. Namespace can have some resources
	// created by the control plane. They must not be deleted.

	return nil
}
