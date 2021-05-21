package resource

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceAccountKind is the name of k8s ServiceAccount resource kind.
const ServiceAccountKind = "ServiceAccount"

// ServiceAccount implements k8s.Resource interface for k8s ServiceAccount.
type ServiceAccount struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
}

// NewServiceAccount returns an initialized ServiceAccount.
func NewServiceAccount(
	c client.Client,
	name, namespace string,
	labels map[string]string) *ServiceAccount {
	return &ServiceAccount{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels: labels,
		client: c,
	}
}

// Get returns an existing ServiceAccount and an error if any.
func (s *ServiceAccount) Get() (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	err := s.client.Get(context.TODO(), s.NamespacedName, sa)
	return sa, err
}

// Create creates a new k8s ServiceAccount resource.
func (s *ServiceAccount) Create() error {
	sa := getServiceAccount(s.Name, s.Namespace, s.labels)
	return Create(s.client, sa)
}

// Delete deletes a ServiceAccount resource.
func (s *ServiceAccount) Delete() error {
	return Delete(s.client, getServiceAccount(s.Name, s.Namespace, s.labels))
}

// getServiceAccount creates a generic service account object.
func getServiceAccount(name, namespace string, labels map[string]string) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIv1,
			Kind:       ServiceAccountKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
