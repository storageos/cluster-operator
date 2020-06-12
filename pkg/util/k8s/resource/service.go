package resource

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceKind is the name of k8s Service resource kind.
const ServiceKind = "Service"

// Service implements k8s.Resource interface for k8s Service resource.
type Service struct {
	types.NamespacedName
	labels      map[string]string
	client      client.Client
	annotations map[string]string
	spec        *corev1.ServiceSpec
}

// NewService returns an initialized Service.
func NewService(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	annotations map[string]string,
	spec *corev1.ServiceSpec) *Service {
	return &Service{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels:      labels,
		client:      c,
		annotations: annotations,
		spec:        spec,
	}
}

// Get returns an existing Service and an error if any.
func (s Service) Get() (*corev1.Service, error) {
	svc := &corev1.Service{}
	err := s.client.Get(context.TODO(), s.NamespacedName, svc)
	return svc, err
}

// Create creates a Service resource.
func (s Service) Create() error {
	svc := getService(s.Name, s.Namespace, s.labels, s.annotations)
	svc.Spec = *s.spec

	// Creating an existing service results in error. Do not fail if the service
	// already exists.
	if err := s.client.Create(context.Background(), svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// Delete deletes a Service resource.
func (s Service) Delete() error {
	return Delete(s.client, getService(s.Name, s.Namespace, s.labels, s.annotations))
}

// getService returns a generic Service object.
func getService(name, namespace string, labels, annotations map[string]string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIv1,
			Kind:       ServiceKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}
