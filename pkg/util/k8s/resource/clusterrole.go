package resource

import (
	"context"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClusterRoleKind is the name of the k8s ClusterRole resource kind.
const ClusterRoleKind = "ClusterRole"

// ClusterRole implements k8s.Resource interface for k8s ClusterRole resource.
type ClusterRole struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	rules  []rbacv1.PolicyRule
}

// NewClusterRole returns an initialized ClusterRole.
func NewClusterRole(
	c client.Client,
	name string,
	labels map[string]string,
	rules []rbacv1.PolicyRule) *ClusterRole {
	return &ClusterRole{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
		labels: labels,
		client: c,
		rules:  rules,
	}
}

// Get returns an existing ClusterRole and an error if any.
func (c ClusterRole) Get() (*rbacv1.ClusterRole, error) {
	cr := &rbacv1.ClusterRole{}
	err := c.client.Get(context.TODO(), c.NamespacedName, cr)
	return cr, err
}

// Create creates a ClusterRole.
func (c ClusterRole) Create() error {
	cr := getClusterRole(c.Name, c.labels)
	cr.Rules = c.rules
	return CreateOrUpdate(c.client, cr)
}

// Delete deletes a ClusterRole resource.
func (c ClusterRole) Delete() error {
	return Delete(c.client, getClusterRole(c.Name, c.labels))
}

// getClusterRole creates a generic ClusterRole object.
func getClusterRole(name string, labels map[string]string) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIrbacv1,
			Kind:       ClusterRoleKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}
