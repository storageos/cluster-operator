// Package k8s provides interfaces, types and functions for k8s related
// utilities.
package k8s

import (
	"github.com/storageos/cluster-operator/pkg/util/k8s/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Resource is an interface for k8s resources. All the k8s resources supported
// by this package must implement this interface.
type Resource interface {
	// Get tries to get an existing resource if any, else returns an error.
	Get() (interface{}, error)

	// Create creates the resource.
	Create() error

	// Delete deletes the resource.
	Delete() error
}

// ResourceManager is k8s resource manager. It provides methods to easily manage
// k8s resources.
type ResourceManager struct {
	client client.Client
	labels map[string]string
}

// TODO: Maybe add a Namespaced ResourceManager to make the namespace of the
// resources implicit.

// NewResourceManager returns an initialized k8s ResourceManager.
func NewResourceManager(client client.Client) *ResourceManager {
	return &ResourceManager{client: client}
}

// SetLabels sets a label for the resources created by the resource manager.
func (r *ResourceManager) SetLabels(labels map[string]string) *ResourceManager {
	if labels == nil {
		labels = map[string]string{}
	}
	r.labels = labels
	return r
}

// ConfigMap returns a ConfigMap object.
// This can also be used to delete an existing object without any references to
// the actual object. The name and namespace, without data, can be used to refer
// the object and perform operations on it.
func (r ResourceManager) ConfigMap(name, namespace string, labels map[string]string, data map[string]string) *resource.ConfigMap {
	return resource.NewConfigMap(r.client, name, namespace, r.combineLabels(labels), data)
}

// DaemonSet returns a DaemonSet object.
func (r ResourceManager) DaemonSet(name, namespace string, labels map[string]string, spec *appsv1.DaemonSetSpec) *resource.DaemonSet {
	return resource.NewDaemonSet(r.client, name, namespace, r.combineLabels(labels), spec)
}

// Deployment returns a Deployment object.
func (r ResourceManager) Deployment(name, namespace string, labels map[string]string, spec *appsv1.DeploymentSpec) *resource.Deployment {
	return resource.NewDeployment(r.client, name, namespace, r.combineLabels(labels), spec)
}

// Ingress returns an Ingress object.
func (r ResourceManager) Ingress(name, namespace string, labels map[string]string, annotations map[string]string, spec *extensionsv1beta1.IngressSpec) *resource.Ingress {
	return resource.NewIngress(r.client, name, namespace, r.combineLabels(labels), annotations, spec)
}

// ServiceAccount returns a ServiceAccount object.
func (r ResourceManager) ServiceAccount(name, namespace string, labels map[string]string) *resource.ServiceAccount {
	return resource.NewServiceAccount(r.client, name, namespace, r.combineLabels(labels))
}

// Role returns a Role object.
func (r ResourceManager) Role(name, namespace string, labels map[string]string, rules []rbacv1.PolicyRule) *resource.Role {
	return resource.NewRole(r.client, name, namespace, r.combineLabels(labels), rules)
}

// RoleBinding returns a RoleBinding object.
func (r ResourceManager) RoleBinding(name, namespace string, labels map[string]string, subjects []rbacv1.Subject, roleRef *rbacv1.RoleRef) *resource.RoleBinding {
	return resource.NewRoleBinding(r.client, name, namespace, r.combineLabels(labels), subjects, roleRef)
}

// ClusterRole returns a ClusterRole object.
func (r ResourceManager) ClusterRole(name string, labels map[string]string, rules []rbacv1.PolicyRule) *resource.ClusterRole {
	return resource.NewClusterRole(r.client, name, r.combineLabels(labels), rules)
}

// ClusterRoleBinding returns a ClusterRoleBinding object.
func (r ResourceManager) ClusterRoleBinding(name string, labels map[string]string, subjects []rbacv1.Subject, roleRef *rbacv1.RoleRef) *resource.ClusterRoleBinding {
	return resource.NewClusterRoleBinding(r.client, name, r.combineLabels(labels), subjects, roleRef)
}

// Secret returns a Secret object.
func (r ResourceManager) Secret(name, namespace string, labels map[string]string, secType corev1.SecretType, data map[string][]byte) *resource.Secret {
	return resource.NewSecret(r.client, name, namespace, r.combineLabels(labels), secType, data)
}

// Service returns a Service object.
func (r ResourceManager) Service(name, namespace string, labels map[string]string, annotations map[string]string, spec *corev1.ServiceSpec) *resource.Service {
	return resource.NewService(r.client, name, namespace, r.combineLabels(labels), annotations, spec)
}

// StatefulSet returns a StatefulSet object.
func (r ResourceManager) StatefulSet(name, namespace string, labels map[string]string, spec *appsv1.StatefulSetSpec) *resource.StatefulSet {
	return resource.NewStatefulSet(r.client, name, namespace, r.combineLabels(labels), spec)
}

// StorageClass returns a StorageClass object.
func (r ResourceManager) StorageClass(name string, labels map[string]string, provisioner string, params map[string]string) *resource.StorageClass {
	return resource.NewStorageClass(r.client, name, r.combineLabels(labels), provisioner, params)
}

// PersistentVolumeClaim returns a PersistentVolumeClaim object.
func (r ResourceManager) PersistentVolumeClaim(name, namespace string, labels map[string]string, spec *corev1.PersistentVolumeClaimSpec) *resource.PVC {
	return resource.NewPVC(r.client, name, namespace, r.combineLabels(labels), spec)
}

func (r ResourceManager) combineLabels(labels map[string]string) map[string]string {
	// Combine the common labels and resource specific labels.
	if labels == nil {
		labels = map[string]string{}
	}
	for k, v := range r.labels {
		labels[k] = v
	}
	return labels
}
