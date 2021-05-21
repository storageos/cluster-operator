package resource

import (
	"context"

	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// IngressKind is the name of k8s Ingress resource kind.
const IngressKind = "Ingress"

// Ingress implements k8s.Resource interface for k8s Ingress resource.
type Ingress struct {
	types.NamespacedName
	labels      map[string]string
	annotations map[string]string
	client      client.Client
	spec        *extensionsv1beta1.IngressSpec
}

// NewIngress returns an initialized Ingress.
func NewIngress(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	annotations map[string]string,
	spec *extensionsv1beta1.IngressSpec) *Ingress {
	return &Ingress{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels:      labels,
		annotations: annotations,
		client:      c,
		spec:        spec,
	}
}

// Get returns an existing Ingress and an error if any.
func (i Ingress) Get() (*extensionsv1beta1.Ingress, error) {
	ingress := &extensionsv1beta1.Ingress{}
	err := i.client.Get(context.TODO(), i.NamespacedName, ingress)
	return ingress, err
}

// Create creates a new Ingress resource.
func (i Ingress) Create() error {
	ingress := getIngress(i.Name, i.Namespace, i.labels, i.annotations)
	ingress.Spec = *i.spec
	return Create(i.client, ingress)
}

// Delete deletes an existing Ingress resource.
func (i Ingress) Delete() error {
	return Delete(i.client, getIngress(i.Name, i.Namespace, i.labels, i.annotations))
}

// getIngress returns a generic Ingress object.
func getIngress(name, namespace string, labels, annotations map[string]string) *extensionsv1beta1.Ingress {
	return &extensionsv1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIextv1beta1,
			Kind:       IngressKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
}
