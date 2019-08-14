package util

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getSecret returns a generic Secret object with the given name and namespace.
func getSecret(name, namespace string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
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

// DeleteSecret deletes a Secret resource, given its name and namespace.
func DeleteSecret(c client.Client, name, namespace string) error {
	return DeleteObject(c, getSecret(name, namespace))
}

// CreateSecret creates a Secret resource, given its name, namespace and data.
func CreateSecret(c client.Client, name, namespace string, secType corev1.SecretType, data map[string][]byte) error {
	secret := getSecret(name, namespace)
	secret.Type = secType
	secret.Data = data
	return CreateOrUpdateObject(c, secret)
}

// CreateCredSecret creates a credential secret of type Opaque.
func CreateCredSecret(c client.Client, name, namespace string, data map[string][]byte) error {
	secType := corev1.SecretType(corev1.SecretTypeOpaque)
	return CreateSecret(c, name, namespace, secType, data)
}

// CreateTLSSecret creates a TLS secret.
func CreateTLSSecret(c client.Client, name, namespace string, data map[string][]byte) error {
	secType := corev1.SecretType(corev1.SecretTypeTLS)
	return CreateSecret(c, name, namespace, secType, data)
}

// GetSecretData fetches the secret data and returns it.
func GetSecretData(c client.Client, name, namespace string) (map[string][]byte, error) {
	secret := getSecret(name, namespace)
	nsName := types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
	if err := c.Get(context.Background(), nsName, secret); err != nil {
		return nil, err
	}
	return secret.Data, nil
}
