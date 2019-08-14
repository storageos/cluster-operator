package util

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getStatefulSet returns a generic StatefulSet object.
func getStatefulSet(name, namespace string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

// DeleteStatefulSet deletes a StatefulSet.
func DeleteStatefulSet(c client.Client, name, namespace string) error {
	return DeleteObject(c, getStatefulSet(name, namespace))
}

// CreateStatefulSet creates a StatefulSet.
func CreateStatefulSet(c client.Client, name, namespace string, spec appsv1.StatefulSetSpec) error {
	statefulset := getStatefulSet(name, namespace)
	statefulset.Spec = spec
	return CreateOrUpdateObject(c, statefulset)
}
