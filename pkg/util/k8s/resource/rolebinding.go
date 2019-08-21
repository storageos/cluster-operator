package resource

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RoleBindingKind is the name of k8s RoleBinding resource kind.
const RoleBindingKind = "RoleBinding"

// RoleBinding implements k8s.Resource interface for k8s RoleBinding resource.
type RoleBinding struct {
	types.NamespacedName
	labels   map[string]string
	client   client.Client
	subjects []rbacv1.Subject
	roleRef  *rbacv1.RoleRef
}

// NewRoleBinding returns an initialized RoleBinding.
func NewRoleBinding(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	subjects []rbacv1.Subject,
	roleRef *rbacv1.RoleRef) *RoleBinding {

	return &RoleBinding{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels:   labels,
		client:   c,
		subjects: subjects,
		roleRef:  roleRef,
	}
}

// Get returns an existing RoleBinding and an error is any.
func (r *RoleBinding) Get() (*rbacv1.RoleBinding, error) {
	roleBinding := &rbacv1.RoleBinding{}
	err := r.client.Get(context.TODO(), r.NamespacedName, roleBinding)
	return roleBinding, err
}

// Create creates a RoleBinding resource.
func (r *RoleBinding) Create() error {
	roleBinding := getRoleBinding(r.Name, r.Namespace, r.labels)
	roleBinding.Subjects = r.subjects
	roleBinding.RoleRef = *r.roleRef
	return CreateOrUpdate(r.client, roleBinding)
}

// Delete deletes a Rolebinding resource.
func (r *RoleBinding) Delete() error {
	return Delete(r.client, getRoleBinding(r.Name, r.Namespace, r.labels))
}

// getRoleBinding returns a generic RoleBinding object.
func getRoleBinding(name, namespace string, labels map[string]string) *rbacv1.RoleBinding {
	return &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIrbacv1,
			Kind:       RoleBindingKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
