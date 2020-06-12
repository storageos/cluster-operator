package resource

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// SecretKind is the name of the k8s Secret resource kind.
const SecretKind = "Secret"

// Secret implements k8s.Resource interface for k8s Secret resource.
type Secret struct {
	types.NamespacedName
	labels  map[string]string
	client  client.Client
	secType corev1.SecretType
	data    map[string][]byte
}

// NewSecret returns an initialized Secret.
func NewSecret(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	secType corev1.SecretType,
	data map[string][]byte) *Secret {
	return &Secret{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels:  labels,
		client:  c,
		secType: secType,
		data:    data,
	}
}

// Get returns an existing Secret and an error if any.
func (s Secret) Get() (*corev1.Secret, error) {
	sec := &corev1.Secret{}
	err := s.client.Get(context.TODO(), s.NamespacedName, sec)
	return sec, err
}

// Delete deletes a Secret resource.
func (s Secret) Delete() error {
	return Delete(s.client, getSecret(s.Name, s.Namespace, s.labels))
}

// Create creates a new Secret resource.
func (s Secret) Create() error {
	secret := getSecret(s.Name, s.Namespace, s.labels)
	secret.Type = s.secType
	secret.Data = s.data
	return CreateOrUpdate(s.client, secret)
}

// getSecret returns a generic Secret object.
func getSecret(name, namespace string, labels map[string]string) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIv1,
			Kind:       SecretKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
