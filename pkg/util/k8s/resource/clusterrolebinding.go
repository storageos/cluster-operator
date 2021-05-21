package resource

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRoleBindingKind is the name of k8s ClusterRoleBinding resource kind.
const ClusterRoleBindingKind = "ClusterRoleBinding"

// ClusterRoleBinding implements k8s.Resource interface for k8s
// ClusterRoleBinding.
type ClusterRoleBinding struct {
	types.NamespacedName
	labels   map[string]string
	client   client.Client
	subjects []rbacv1.Subject
	roleRef  *rbacv1.RoleRef
}

// NewClusterRoleBinding returns an initialized ClusterRoleBinding.
func NewClusterRoleBinding(
	c client.Client,
	name string,
	labels map[string]string,
	subjects []rbacv1.Subject,
	roleRef *rbacv1.RoleRef) *ClusterRoleBinding {
	return &ClusterRoleBinding{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
		labels:   labels,
		client:   c,
		subjects: subjects,
		roleRef:  roleRef,
	}
}

// Get returns an existing ClusterRoleBinding and an error if any.
func (c ClusterRoleBinding) Get() (*rbacv1.ClusterRoleBinding, error) {
	crb := &rbacv1.ClusterRoleBinding{}
	err := c.client.Get(context.TODO(), c.NamespacedName, crb)
	return crb, err
}

// Create creates a ClusterRoleBinding.
func (c ClusterRoleBinding) Create() error {
	roleBinding := getClusterRoleBinding(c.Name, c.labels)
	roleBinding.Subjects = c.subjects
	roleBinding.RoleRef = *c.roleRef
	return Create(c.client, roleBinding)
}

// Delete deletes a ClusterRoleBinding resources.
func (c ClusterRoleBinding) Delete() error {
	return Delete(c.client, getClusterRoleBinding(c.Name, c.labels))
}

// getClusterRoleBinding returns a generic ClusterRoleBinding object.
func getClusterRoleBinding(name string, labels map[string]string) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIrbacv1,
			Kind:       ClusterRoleBindingKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}
