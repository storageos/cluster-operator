package util

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getConfigMap returns an empty ConfigMap object. This can be used while
// creating a configmap resource.
func getConfigMap(name, namespace string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

// DeleteConfigMap deletes a ConfigMap, given its name and namespace.
func DeleteConfigMap(c client.Client, name, namespace string) error {
	return DeleteObject(c, getConfigMap(name, namespace))
}

// CreateConfigMap creates a ConfigMap, given its name, namespace, and data.
func CreateConfigMap(c client.Client, name, namespace string, data map[string]string) error {
	configmap := getConfigMap(name, namespace)
	configmap.Data = data
	return CreateOrUpdateObject(c, configmap)
}
