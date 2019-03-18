package storageos

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s *Deployment) createServiceAccount(name string) error {
	sa := s.getServiceAccount(name)
	return s.createOrUpdateObject(sa)
}

func (s *Deployment) deleteServiceAccount(name string) error {
	return s.deleteObject(s.getServiceAccount(name))
}

// getServiceAccount creates a generic service account object with the given
// name and returns it.
func (s *Deployment) getServiceAccount(name string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

func (s *Deployment) createServiceAccountForDaemonSet() error {
	return s.createServiceAccount("storageos-daemonset-sa")
}

func (s *Deployment) createServiceAccountForStatefulSet() error {
	return s.createServiceAccount("storageos-statefulset-sa")
}

func (s *Deployment) createRoleForKeyMgmt() error {
	role := s.getRole(keyManagementRoleName)
	role.Rules = []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"secrets"},
			Verbs:     []string{"get", "list", "create", "delete"},
		},
	}

	return s.createOrUpdateObject(role)
}

func (s *Deployment) deleteRole(name string) error {
	return s.deleteObject(s.getRole(keyManagementRoleName))
}

// getRole creates a generic role object with the given name and returns it.
func (s *Deployment) getRole(name string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

func (s *Deployment) createClusterRole(name string, rules []rbacv1.PolicyRule) error {
	role := s.getClusterRole(name)
	role.Rules = rules
	return s.createOrUpdateObject(role)
}

func (s *Deployment) deleteClusterRole(name string) error {
	return s.deleteObject(s.getClusterRole(name))
}

func (s *Deployment) getClusterRole(name string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
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
	return s.createClusterRole("driver-registrar-role", rules)
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
	return s.createClusterRole("csi-provisioner-role", rules)
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
	return s.createClusterRole("csi-attacher-role", rules)
}

func (s *Deployment) createRoleBindingForKeyMgmt() error {
	roleBinding := s.getRoleBinding(keyManagementBindingName)
	roleBinding.Subjects = []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-daemonset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleBinding.RoleRef = rbacv1.RoleRef{
		Kind:     "Role",
		Name:     keyManagementRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createOrUpdateObject(roleBinding)
}

func (s *Deployment) deleteRoleBinding(name string) error {
	return s.deleteObject(s.getRoleBinding(name))
}

func (s *Deployment) getRoleBinding(name string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

func (s *Deployment) createClusterRoleBinding(name string, subjects []rbacv1.Subject, roleRef rbacv1.RoleRef) error {
	roleBinding := s.getClusterRoleBinding(name)
	roleBinding.Subjects = subjects
	roleBinding.RoleRef = roleRef
	return s.createOrUpdateObject(roleBinding)
}

func (s *Deployment) deleteClusterRoleBinding(name string) error {
	return s.deleteObject(s.getClusterRoleBinding(name))
}

func (s *Deployment) getClusterRoleBinding(name string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

func (s *Deployment) createClusterRoleBindingForDriverRegistrar() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-daemonset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "driver-registrar-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createClusterRoleBinding("driver-registrar-binding", subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForK8SDriverRegistrar() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-statefulset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "driver-registrar-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createClusterRoleBinding("k8s-driver-registrar-binding", subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForProvisioner() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-statefulset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "csi-provisioner-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createClusterRoleBinding("csi-provisioner-binding", subjects, roleRef)
}

func (s *Deployment) createClusterRoleBindingForAttacher() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-statefulset-sa",
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "csi-attacher-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	return s.createClusterRoleBinding("csi-attacher-binding", subjects, roleRef)
}
