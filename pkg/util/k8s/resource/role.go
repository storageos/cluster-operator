package resource

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// RoleKind is the name of k8s Role resource kind.
const RoleKind = "Role"

// Role implements k8s.Resource interface for k8s Role resource.
type Role struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	rules  []rbacv1.PolicyRule
}

// NewRole returns an initialized Role.
func NewRole(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	rules []rbacv1.PolicyRule) *Role {
	return &Role{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels: labels,
		client: c,
		rules:  rules,
	}
}

// Get returns an existing Role and an error if any.
func (r Role) Get() (*rbacv1.Role, error) {
	role := &rbacv1.Role{}
	err := r.client.Get(context.TODO(), r.NamespacedName, role)
	return role, err
}

// Create creates a k8s Role resource.
func (r *Role) Create() error {
	role := getRole(r.Name, r.Namespace, r.labels)
	role.Rules = r.rules
	return Create(r.client, role)
}

// Delete deletes a k8s Role resource.
func (r *Role) Delete() error {
	return Delete(r.client, getRole(r.Name, r.Namespace, r.labels))
}

// getRole creates a generic Role object.
func getRole(name, namespace string, labels map[string]string) *rbacv1.Role {
	return &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIrbacv1,
			Kind:       RoleKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
