package resource

import (
	"context"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ServiceMonitorKind is the name of the k8s ServiceMonitor resource kind.
const ServiceMonitorKind = "ServiceMonitor"

// ServiceMonitor implements k8s.Resource interface for k8s ServiceMonitor resource.
type ServiceMonitor struct {
	types.NamespacedName
	labels      map[string]string
	client      client.Client
	annotations map[string]string
	service     *corev1.Service
	spec        *monitoringv1.ServiceMonitorSpec
}

// NewServiceMonitor returns an initialized ServiceMonitor.
func NewServiceMonitor(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	annotations map[string]string,
	svc *corev1.Service,
	spec *monitoringv1.ServiceMonitorSpec) *ServiceMonitor {
	return &ServiceMonitor{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels:      labels,
		client:      c,
		annotations: annotations,
		service:     svc,
		spec:        spec,
	}
}

// Get returns an existing ServiceMonitor and an error if any.
func (s ServiceMonitor) Get() (*monitoringv1.ServiceMonitor, error) {
	sm := &monitoringv1.ServiceMonitor{}
	err := s.client.Get(context.TODO(), s.NamespacedName, sm)
	return sm, err
}

// Create creates a ServiceMonitor.
func (s ServiceMonitor) Create() error {
	sm := getServiceMonitor(s.Name, s.Namespace, s.labels, s.annotations, s.service)
	sm.Spec = *s.spec
	return CreateOrUpdate(s.client, sm)
}

// Delete deletes a ServiceMonitor.
func (s ServiceMonitor) Delete() error {
	return Delete(s.client, getServiceMonitor(s.Name, s.Namespace, s.labels, s.annotations, s.service))
}

// getServiceMonitor returns a generic ServiceMonitor object.
func getServiceMonitor(name, namespace string, labels map[string]string, annotations map[string]string, svc *corev1.Service) *monitoringv1.ServiceMonitor {
	boolTrue := true
	ownerRef := metav1.OwnerReference{
		APIVersion:         "v1",
		BlockOwnerDeletion: &boolTrue,
		Controller:         &boolTrue,
		Kind:               "Service",
		Name:               svc.Name,
		UID:                svc.UID,
	}
	return &monitoringv1.ServiceMonitor{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIservicemonitorv1,
			Kind:       ServiceMonitorKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			Labels:          labels,
			Annotations:     annotations,
			OwnerReferences: []metav1.OwnerReference{ownerRef},
		},
	}
}
