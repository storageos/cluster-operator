package storageos

import (
	rbacv1 "k8s.io/api/rbac/v1"
)

// Exported role, binding and service account resource names.
const (
	DaemonsetSA   = "storageos-daemonset-sa"
	StatefulsetSA = "storageos-statefulset-sa"
	CSIHelperSA   = "storageos-csi-helper-sa"
	SchedulerSA   = "storageos-scheduler-sa"
	APIManagerSA  = "storageos-api-manager-sa"

	CSIProvisionerClusterRoleName    = "storageos:csi-provisioner"
	CSIProvisionerClusterBindingName = "storageos:csi-provisioner"

	CSIAttacherClusterRoleName    = "storageos:csi-attacher"
	CSIAttacherClusterBindingName = "storageos:csi-attacher"

	CSIResizerClusterRoleName    = "storageos:csi-resizer"
	CSIResizerClusterBindingName = "storageos:csi-resizer"

	CSIDriverRegistrarClusterRoleName       = "storageos:driver-registrar"
	CSIDriverRegistrarClusterBindingName    = "storageos:driver-registrar"
	CSIK8SDriverRegistrarClusterBindingName = "storageos:k8s-driver-registrar"

	// OpenShift Security Context Constraints role and role binding names.
	OpenShiftSCCClusterRoleName    = "storageos:openshift-scc"
	OpenShiftSCCClusterBindingName = "storageos:openshift-scc"

	KeyManagementClusterRoleName    = "storageos:key-management"
	KeyManagementClusterBindingName = "storageos:key-management"

	FencingClusterRoleName    = "storageos:pod-fencer"
	FencingClusterBindingName = "storageos:pod-fencer"

	NFSClusterRoleName    = "storageos:nfs-provisioner"
	NFSClusterBindingName = "storageos:nfs-provisioner"

	KubeSchedulerClusterRoleName    = "storageos:kube-scheduler"
	KubeSchedulerClusterBindingName = "storageos:kube-scheduler"

	SchedulerExtenderClusterRoleName    = "storageos:scheduler-extender"
	SchedulerExtenderClusterBindingName = "storageos:scheduler-extender"

	InitClusterRoleName    = "storageos:init"
	InitClusterBindingName = "storageos:init"

	APIManagerClusterRoleName    = "storageos:api-manager"
	APIManagerClusterBindingName = "storageos:api-manager"
)

// getCSIHelperServiceAccountName returns the service account name of CSI helper
// based on the cluster configuration.
func (s *Deployment) getCSIHelperServiceAccountName() string {
	switch s.stos.Spec.GetCSIDeploymentStrategy() {
	case deploymentKind:
		return CSIHelperSA
	default:
		return StatefulsetSA
	}
}

// createServiceAccountForDaemonSet creates a service account fot the DaemonSet
// pods.
func (s *Deployment) createServiceAccountForDaemonSet() error {
	return s.k8sResourceManager.ServiceAccount(DaemonsetSA, s.stos.Spec.GetResourceNS(), nil).Create()
}

// createServiceAccountForCSIHelper creates service account for the appropriate
// CSI helper kind based on the cluster config.
func (s *Deployment) createServiceAccountForCSIHelper() error {
	return s.k8sResourceManager.ServiceAccount(s.getCSIHelperServiceAccountName(), s.stos.Spec.GetResourceNS(), nil).Create()
}

// createServiceAccountForScheduler creates a service account for scheduler
// extender.
func (s *Deployment) createServiceAccountForScheduler() error {
	return s.k8sResourceManager.ServiceAccount(SchedulerSA, s.stos.Spec.GetResourceNS(), nil).Create()
}

// createServiceAccountForAPIManager creates a service account for api-manager.
func (s *Deployment) createServiceAccountForAPIManager() error {
	return s.k8sResourceManager.ServiceAccount(APIManagerSA, s.stos.Spec.GetResourceNS(), nil).Create()
}

func (s *Deployment) createClusterRoleForKeyMgmt() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"create", "delete", "get", "list", "patch", "update", "watch"},
		},
	}
	return s.k8sResourceManager.ClusterRole(KeyManagementClusterRoleName, nil, rules).Create()
}

// createClusterRoleForAPIManager creates a role with all the permissions for
// api-manager.
func (s *Deployment) createClusterRoleForAPIManager() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"endpoints",
				"services",
				"configmaps",
			},
			Verbs: []string{"create", "delete", "get", "list", "patch", "update", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{
				"endpoints/status",
				"services/status",
				"configmaps/status",
			},
			Verbs: []string{"get", "patch", "update"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{
				"namespaces",
				"nodes",
				"persistentvolumeclaims",
			},
			Verbs: []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list", "watch", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
			Verbs:     []string{"get", "create", "update"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"volumeattachments"},
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"admissionregistration.k8s.io"},
			Resources: []string{"mutatingwebhookconfigurations"},
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete"},
		},
	}
	return s.k8sResourceManager.ClusterRole(APIManagerClusterRoleName, nil, rules).Create()
}

func (s *Deployment) createClusterRoleForFencing() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list", "watch", "update", "patch", "delete"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"volumeattachments"},
			Verbs:     []string{"get", "list", "watch", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	return s.k8sResourceManager.ClusterRole(FencingClusterRoleName, nil, rules).Create()
}

func (s *Deployment) createClusterRoleForNFS() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"storageos.com"},
			Resources: []string{"nfsservers"},
			Verbs:     []string{"get", "list", "watch", "create", "update", "patch", "delete"},
		},
	}
	return s.k8sResourceManager.ClusterRole(NFSClusterRoleName, nil, rules).Create()
}

func (s *Deployment) createClusterRoleForDriverRegistrar() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "update"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
		{
			APIGroups: []string{"apiextensions.k8s.io"},
			Resources: []string{"customresourcedefinitions"},
			Verbs:     []string{"create"},
		},
		{
			APIGroups: []string{"csi.storage.k8s.io"},
			Resources: []string{"csidrivers"},
			Verbs:     []string{"create"},
		},
	}
	return s.k8sResourceManager.ClusterRole(CSIDriverRegistrarClusterRoleName, nil, rules).Create()
}

func (s *Deployment) createClusterRoleForProvisioner() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"get", "list", "watch", "create", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"get", "list", "watch", "update"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses", "csinodes"},
			Verbs:     []string{"list", "watch", "get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"list", "watch", "get"},
		},
	}
	return s.k8sResourceManager.ClusterRole(CSIProvisionerClusterRoleName, nil, rules).Create()
}

func (s *Deployment) createClusterRoleForAttacher() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"list", "watch", "get"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"volumeattachments"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"volumeattachments/status"},
			Verbs:     []string{"get", "patch", "update"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"csinodeinfos", "csinodes"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	return s.k8sResourceManager.ClusterRole(CSIAttacherClusterRoleName, nil, rules).Create()
}

func (s *Deployment) createClusterRoleForResizer() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"get", "list", "watch", "update", "patch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims/status"},
			Verbs:     []string{"update", "patch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	return s.k8sResourceManager.ClusterRole(CSIResizerClusterRoleName, nil, rules).Create()
}

// createClusterRoleForKubeScheduler creates a ClusterRole resource for scheduler
// extender with all the permissions required by kube-scheduler.
func (s *Deployment) createClusterRoleForKubeScheduler() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"configmaps",
				"persistentvolumes",
				"persistentvolumeclaims",
				"nodes",
				"replicationcontrollers",
				"pods",
				"pods/binding",
				"pods/status",
				"services",
				"endpoints",
				"events",
			},
			Verbs: []string{"get", "list", "watch", "create", "update", "patch"},
		},
		{
			APIGroups: []string{"apps"},
			Resources: []string{"statefulsets", "replicasets"},
			Verbs:     []string{"list", "watch"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses", "csinodes"},
			Verbs:     []string{"list", "watch"},
		},
		{
			APIGroups: []string{"policy"},
			Resources: []string{"poddisruptionbudgets"},
			Verbs:     []string{"list", "watch"},
		},
		{
			APIGroups: []string{"events.k8s.io"},
			Resources: []string{"events"},
			Verbs:     []string{"create", "patch"},
		},
		{
			APIGroups: []string{"coordination.k8s.io"},
			Resources: []string{"leases"},
			Verbs:     []string{"get", "create", "update"},
		},
	}
	return s.k8sResourceManager.ClusterRole(KubeSchedulerClusterRoleName, nil, rules).Create()
}

// createClusterRoleForSchedulerExtender creates a ClusterRole resource for scheduler
// extender with all the permissions required by custom scheduler extender.
func (s *Deployment) createClusterRoleForSchedulerExtender() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{
				"persistentvolumes",
				"persistentvolumeclaims",
			},
			Verbs: []string{"get"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"get"},
		},
	}
	return s.k8sResourceManager.ClusterRole(SchedulerExtenderClusterRoleName, nil, rules).Create()
}

func (s *Deployment) createClusterRoleBindingForKeyMgmt() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
		{
			Kind:      "ServiceAccount",
			Name:      APIManagerSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     KeyManagementClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(KeyManagementClusterBindingName, nil, subjects, roleRef).Create()
}

// createClusterRoleBindingForAPIManager creates a role binding for api-manager.
func (s *Deployment) createClusterRoleBindingForAPIManager() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      APIManagerSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     APIManagerClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(APIManagerClusterBindingName, nil, subjects, roleRef).Create()
}

func (s *Deployment) createClusterRoleBindingForFencing() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     FencingClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(FencingClusterBindingName, nil, subjects, roleRef).Create()
}

func (s *Deployment) createClusterRoleBindingForNFS() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     NFSClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(NFSClusterBindingName, nil, subjects, roleRef).Create()
}

func (s *Deployment) createClusterRoleBindingForDriverRegistrar() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIDriverRegistrarClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(CSIDriverRegistrarClusterBindingName, nil, subjects, roleRef).Create()
}

func (s *Deployment) createClusterRoleBindingForK8SDriverRegistrar() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIDriverRegistrarClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(CSIK8SDriverRegistrarClusterBindingName, nil, subjects, roleRef).Create()
}

func (s *Deployment) createClusterRoleBindingForProvisioner() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIProvisionerClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(CSIProvisionerClusterBindingName, nil, subjects, roleRef).Create()
}

func (s *Deployment) createClusterRoleBindingForAttacher() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIAttacherClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(CSIAttacherClusterBindingName, nil, subjects, roleRef).Create()
}

func (s *Deployment) createClusterRoleBindingForResizer() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIResizerClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(CSIResizerClusterBindingName, nil, subjects, roleRef).Create()
}

// createClusterRoleForSCC creates cluster role with api group and resource
// specific to openshift. This permission is required for by daemonsets and
// statefulsets.
func (s *Deployment) createClusterRoleForSCC() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{"security.openshift.io"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
			ResourceNames: []string{"privileged"},
		},
	}
	return s.k8sResourceManager.ClusterRole(OpenShiftSCCClusterRoleName, nil, rules).Create()
}

// createClusterRoleBindingForSCC creates a cluster role binding of the
// openshift SCC role with daemonset and statefulset service account.
func (s *Deployment) createClusterRoleBindingForSCC() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}

	// Add Statefulset service account if CSI is enabled.
	if s.stos.Spec.CSI.Enable {
		subjects = append(subjects, rbacv1.Subject{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		})
	}

	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     OpenShiftSCCClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(OpenShiftSCCClusterBindingName, nil, subjects, roleRef).Create()
}

// createClusterRoleBindingForKubeScheduler creates a cluster role binding for the
// kube-scheduler.
func (s *Deployment) createClusterRoleBindingForKubeScheduler() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      SchedulerSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     KubeSchedulerClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(KubeSchedulerClusterBindingName, nil, subjects, roleRef).Create()
}

// createClusterRoleBindingForScheduler creates a cluster role binding for the
// custom scheduler extender.
func (s *Deployment) createClusterRoleBindingForSchedulerExtender() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     SchedulerExtenderClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(SchedulerExtenderClusterBindingName, nil, subjects, roleRef).Create()
}

// createClusterRoleForInit creates cluster role for the init container. This is
// needed by the init container to fetch StorageOS DaemonSet and get the current
// StorageOS node image.
func (s *Deployment) createClusterRoleForInit() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"apps"},
			Resources: []string{"daemonsets"},
			Verbs:     []string{"get"},
		},
	}
	return s.k8sResourceManager.ClusterRole(InitClusterRoleName, nil, rules).Create()
}

// createClusterRoleBindingForInit creates a cluster role binding of the init
// container role with daemonset service account.
func (s *Deployment) createClusterRoleBindingForInit() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     InitClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.k8sResourceManager.ClusterRoleBinding(InitClusterBindingName, nil, subjects, roleRef).Create()
}
