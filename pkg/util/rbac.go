package util

import (
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	appName = "storageos"
)

// getServiceAccount creates a generic service account object with the given
// name and namespace, and returns it.
func getServiceAccount(name, namespace string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

// DeleteServiceAccount deletes a ServiceAccount resource, given its name and
// namespace.
func DeleteServiceAccount(c client.Client, name, namespace string) error {
	return DeleteObject(c, getServiceAccount(name, namespace))
}

// CreateServiceAccount creates a ServiceAccount resource, given name and
// namespace.
func CreateServiceAccount(c client.Client, name, namespace string) error {
	sa := getServiceAccount(name, namespace)
	return CreateOrUpdateObject(c, sa)
}

// getRole creates a generic role object with the given name and namespace, and
// returns it.
func getRole(name, namespace string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

// DeleteRole deletes a Role resource, given its name and namespace.
func DeleteRole(c client.Client, name, namespace string) error {
	return DeleteObject(c, getRole(name, namespace))
}

// CreateRole creates a Role, given name, namespace and rules.
func CreateRole(c client.Client, name, namespace string, rules []rbacv1.PolicyRule) error {
	role := getRole(name, namespace)
	role.Rules = rules
	return CreateOrUpdateObject(c, role)
}

func getRoleBinding(name, namespace string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

// DeleteRoleBinding deletes a Rolebinding resource, given its name and
// namespace.
func DeleteRoleBinding(c client.Client, name, namespace string) error {
	return DeleteObject(c, getRoleBinding(name, namespace))
}

// CreateRoleBinding creates a RoleBinding
func CreateRoleBinding(c client.Client, name, namespace string, subjects []rbacv1.Subject, roleRef rbacv1.RoleRef) error {
	roleBinding := getRoleBinding(name, namespace)
	roleBinding.Subjects = subjects
	roleBinding.RoleRef = roleRef
	return CreateOrUpdateObject(c, roleBinding)
}

// getClusterRole creates a generic ClusterRole object with the given name and
// returns it.
func getClusterRole(name string) *rbacv1.ClusterRole {
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

// DeleteClusterRole deletes a ClusterRole resource, given its name.
func DeleteClusterRole(c client.Client, name string) error {
	return DeleteObject(c, getClusterRole(name))
}

// CreateClusterRole creates a ClusterRole, given name and rules.
func CreateClusterRole(c client.Client, name string, rules []rbacv1.PolicyRule) error {
	role := getClusterRole(name)
	role.Rules = rules
	return CreateOrUpdateObject(c, role)
}

// getClusterRoleBinding returns a ClusterRoleBinding object, given a name.
// This can be used for creation or deletion of a resource by name or get a
// ClusterRoleBinding resource template.
func getClusterRoleBinding(name string) *rbacv1.ClusterRoleBinding {
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

// DeleteClusterRoleBinding deletes a ClusterRoleBinding resources, given its
// name.
func DeleteClusterRoleBinding(c client.Client, name string) error {
	return DeleteObject(c, getClusterRoleBinding(name))
}

// CreateClusterRoleBinding creates a ClusterRoleBinding, given name, subject and
// role ref.
func CreateClusterRoleBinding(c client.Client, name string, subjects []rbacv1.Subject, roleRef rbacv1.RoleRef) error {
	roleBinding := getClusterRoleBinding(name)
	roleBinding.Subjects = subjects
	roleBinding.RoleRef = roleRef
	return CreateOrUpdateObject(c, roleBinding)
}
