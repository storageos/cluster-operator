package util

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getDaemonSet returns a generic DaemonSet object.
func getDaemonSet(name, namespace string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
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

// DeleteDaemonSet deletes a DaemonSet resource.
func DeleteDaemonSet(c client.Client, name, namespace string) error {
	return DeleteObject(c, getDaemonSet(name, namespace))
}

// CreateDaemonSet creates a DaemonSet resource.
func CreateDaemonSet(c client.Client, name, namespace string, spec appsv1.DaemonSetSpec) error {
	daemonset := getDaemonSet(name, namespace)
	daemonset.Spec = spec
	return CreateOrUpdateObject(c, daemonset)
}
