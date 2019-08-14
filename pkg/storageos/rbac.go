package storageos

import (
	"github.com/storageos/cluster-operator/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
)

// Exported role, binding and service account resource names.
const (
	DaemonsetSA   = "storageos-daemonset-sa"
	StatefulsetSA = "storageos-statefulset-sa"
	CSIHelperSA   = "storageos-csi-helper-sa"
	SchedulerSA   = "storageos-scheduler-sa"

	CSIProvisionerClusterRoleName    = "storageos:csi-provisioner"
	CSIProvisionerClusterBindingName = "storageos:csi-provisioner"

	CSIAttacherClusterRoleName    = "storageos:csi-attacher"
	CSIAttacherClusterBindingName = "storageos:csi-attacher"

	CSIDriverRegistrarClusterRoleName       = "storageos:driver-registrar"
	CSIDriverRegistrarClusterBindingName    = "storageos:driver-registrar"
	CSIK8SDriverRegistrarClusterBindingName = "storageos:k8s-driver-registrar"

	// OpenShift Security Context Constraints role and role binding names.
	OpenShiftSCCClusterRoleName    = "storageos:openshift-scc"
	OpenShiftSCCClusterBindingName = "storageos:openshift-scc"

	KeyManagementRoleName    = "storageos:key-management"
	KeyManagementBindingName = "storageos:key-management"

	FencingClusterRoleName    = "storageos:pod-fencer"
	FencingClusterBindingName = "storageos:pod-fencer"

	SchedulerClusterRoleName    = "storageos:scheduler-extender"
	SchedulerClusterBindingName = "storageos:scheduler-extender"
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
	return util.CreateServiceAccount(s.client, DaemonsetSA, s.stos.Spec.GetResourceNS())
}

// createServiceAccountForCSIHelper creates service account for the appropriate
// CSI helper kind based on the cluster config.
func (s *Deployment) createServiceAccountForCSIHelper() error {
	return util.CreateServiceAccount(s.client, s.getCSIHelperServiceAccountName(), s.stos.Spec.GetResourceNS())
}

// createServiceAccountForScheduler creates a service account for scheduler
// extender.
func (s *Deployment) createServiceAccountForScheduler() error {
	return util.CreateServiceAccount(s.client, SchedulerSA, s.stos.Spec.GetResourceNS())
}

func (s *Deployment) createRoleForKeyMgmt() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get", "list", "create", "delete"},
		},
	}
	return util.CreateRole(s.client, KeyManagementRoleName, s.stos.Spec.GetResourceNS(), rules)
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
	return util.CreateClusterRole(s.client, FencingClusterRoleName, rules)
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
	return util.CreateClusterRole(s.client, CSIDriverRegistrarClusterRoleName, rules)
}

func (s *Deployment) createClusterRoleForProvisioner() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"list", "watch", "create", "delete"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"get", "list", "watch", "update"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
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
	}
	return util.CreateClusterRole(s.client, CSIProvisionerClusterRoleName, rules)
}

func (s *Deployment) createClusterRoleForAttacher() error {
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumes"},
			Verbs:     []string{"get", "list", "watch", "update"},
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
			Verbs:     []string{"get", "list", "watch", "update"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"csinodeinfos"},
			Verbs:     []string{"get", "list", "watch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	return util.CreateClusterRole(s.client, CSIAttacherClusterRoleName, rules)
}

// createClusterRoleForScheduler creates a ClusterRole resource for scheduler
// extender with all the permissions required by kube-scheduler.
func (s *Deployment) createClusterRoleForScheduler() error {
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
			Resources: []string{"storageclasses"},
			Verbs:     []string{"list", "watch"},
		},
		{
			APIGroups: []string{"policy"},
			Resources: []string{"poddisruptionbudgets"},
			Verbs:     []string{"list", "watch"},
		},
	}
	return util.CreateClusterRole(s.client, SchedulerClusterRoleName, rules)
}

func (s *Deployment) createRoleBindingForKeyMgmt() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "Role",
		Name:     KeyManagementRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateRoleBinding(s.client, KeyManagementBindingName, s.stos.Spec.GetResourceNS(), subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForFencing() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     FencingClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(s.client, FencingClusterBindingName, subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForDriverRegistrar() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      DaemonsetSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIDriverRegistrarClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(s.client, CSIDriverRegistrarClusterBindingName, subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForK8SDriverRegistrar() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIDriverRegistrarClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(s.client, CSIK8SDriverRegistrarClusterBindingName, subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForProvisioner() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIProvisionerClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(s.client, CSIProvisionerClusterBindingName, subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForAttacher() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      s.getCSIHelperServiceAccountName(),
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     CSIAttacherClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(s.client, CSIAttacherClusterBindingName, subjects, roleRef)
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
	return util.CreateClusterRole(s.client, OpenShiftSCCClusterRoleName, rules)
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

	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     OpenShiftSCCClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(s.client, OpenShiftSCCClusterBindingName, subjects, roleRef)
}

// createClusterRoleBindingForScheduler creates a cluster role binding for the
// scheduler extender.
func (s *Deployment) createClusterRoleBindingForScheduler() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      SchedulerSA,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     SchedulerClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(s.client, SchedulerClusterBindingName, subjects, roleRef)
}
