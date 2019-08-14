package util

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getDeployment returns a generic Deployment object given the name and
// namespace.
func getDeployment(name, namespace string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": "storageos",
			},
		},
	}
}

// DeleteDeployment deletes a Deployment resource, given a name and namespace.
func DeleteDeployment(c client.Client, name, namespace string) error {
	return DeleteObject(c, getDeployment(name, namespace))
}

// CreateDeployment creates a Deployment resource, given name, namespace and
// deployment spec.
func CreateDeployment(c client.Client, name, namespace string, spec appsv1.DeploymentSpec) error {
	deployment := getDeployment(name, namespace)
	deployment.Spec = spec
	return CreateOrUpdateObject(c, deployment)
}
