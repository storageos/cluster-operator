package util

import (
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getIngress returns a generic Ingress object.
func getIngress(name, namespace string, annotations map[string]string) *extensionsv1beta1.Ingress {
	return &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
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

// DeleteIngress deletes an Ingress resource.
func DeleteIngress(c client.Client, name, namespace string) error {
	return DeleteObject(c, getIngress(name, namespace, nil))
}

// CreateIngress creates an Ingress resource.
func CreateIngress(c client.Client, name, namespace string, annotations map[string]string, spec extensionsv1beta1.IngressSpec) error {
	ingress := getIngress(name, namespace, annotations)
	ingress.Spec = spec
	return CreateOrUpdateObject(c, ingress)
}
