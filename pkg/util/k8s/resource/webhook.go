package resource

import (
	"context"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MutatingWebhookConfigurationKind is the name of the k8s MutatingWebhookConfiguration resource kind.
const MutatingWebhookConfigurationKind = "MutatingWebhookConfiguration"

// MutatingWebhookConfiguration implements k8s.Resource interface for the k8s
// MutatingWebhookConfiguration resource.
type MutatingWebhookConfiguration struct {
	types.NamespacedName
	labels   map[string]string
	client   client.Client
	webhooks []admissionv1.MutatingWebhook
}

// NewMutatingWebhookConfiguration returns an initialized MutatingWebhookConfiguration.
func NewMutatingWebhookConfiguration(
	c client.Client,
	name string,
	labels map[string]string,
	webhooks []admissionv1.MutatingWebhook) *MutatingWebhookConfiguration {
	return &MutatingWebhookConfiguration{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
		labels:   labels,
		client:   c,
		webhooks: webhooks,
	}
}

// Get returns an existing MutatingWebhookConfiguration and an error if any.
func (w MutatingWebhookConfiguration) Get() (*admissionv1.MutatingWebhookConfiguration, error) {
	wh := &admissionv1.MutatingWebhookConfiguration{}
	err := w.client.Get(context.TODO(), w.NamespacedName, wh)
	return wh, err
}

// Create creates a MutatingWebhookConfiguration.
func (w MutatingWebhookConfiguration) Create() error {
	wh := getMutatingWebhookConfiguration(w.Name, w.labels, w.webhooks)
	return CreateOrUpdate(w.client, wh)
}

// Delete deletes a MutatingWebhookConfiguration.
func (w MutatingWebhookConfiguration) Delete() error {
	return Delete(w.client, getMutatingWebhookConfiguration(w.Name, w.labels, w.webhooks))
}

// getMutatingWebhookConfiguration returns a generic MutatingWebhookConfiguration object.
func getMutatingWebhookConfiguration(name string, labels map[string]string, webhooks []admissionv1.MutatingWebhook) *admissionv1.MutatingWebhookConfiguration {
	return &admissionv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIadmissionv1,
			Kind:       MutatingWebhookConfigurationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Webhooks: webhooks,
	}
}
