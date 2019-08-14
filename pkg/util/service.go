package util

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getService returns a generic Service object with the given name, namespace
// and annotations.
func getService(name, namespace string, annotations map[string]string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": appName,
			},
			Annotations: annotations,
		},
	}
}

// DeleteService deletes a Service resource, given its name and namespace.
func DeleteService(c client.Client, name, namespace string) error {
	return DeleteObject(c, getService(name, namespace, nil))
}

// CreateService creates a Service resource, given its name, namespace and
// annotations.
func CreateService(c client.Client, name, namespace string, annotations map[string]string, spec corev1.ServiceSpec) error {
	svc := getService(name, namespace, annotations)
	svc.Spec = spec
	// return CreateOrUpdateObject(c, svc)

	// Creating an existing service results in error. Do not fail if the service
	// already exists.
	if err := c.Create(context.Background(), svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}
