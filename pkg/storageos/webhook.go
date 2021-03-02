package storageos

import (
	"github.com/storageos/cluster-operator/pkg/util/k8s"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	// MutatingWebhookConfigName is the name used for the mutating webhook
	// configuration.
	MutatingWebhookConfigName = "storageos-mutating-webhook"

	// WebhookServiceName name is the name of the service that will be created
	// for the webhook.
	WebhookServiceName = "storageos-webhook"

	// ServicePort configuration.
	webhookPortName         = "webhooks"
	webhookPort       int32 = 443
	webhookTargetPort int32 = 9443

	// webhookPodMutatorName is the name of the webhook configuration
	webhookPodMutatorName = "pod-mutator.storageos.com"
)

// createWebhookConfiguration creates the webhook service and configuration.
func (s *Deployment) createWebhookConfiguration() error {
	if err := s.createWebhookService(); err != nil {
		return err
	}

	return s.createMutatingWebhookConfiguration()
}

// createWebhookService creates a Service for the api-manager webhooks listener.
func (s Deployment) createWebhookService() error {
	spec := &corev1.ServiceSpec{
		Type: corev1.ServiceType(s.stos.Spec.GetServiceType()),
		Ports: []corev1.ServicePort{
			{
				Name:       webhookPortName,
				Protocol:   "TCP",
				Port:       webhookPort,
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: webhookTargetPort},
			},
		},
		Selector: map[string]string{
			k8s.AppComponent: APIManagerName,
		},
	}
	labels := podLabelsForAPIManager(s.stos.Name)

	return s.k8sResourceManager.Service(WebhookServiceName, s.stos.Spec.GetResourceNS(), labels, nil, spec).Create()
}

// createMutatingWebhookConfiguration creates the configuration for the mutating
// webhooks.  The configuration will be updated by the api-manager to include
// the CA cert bundle.
func (s Deployment) createMutatingWebhookConfiguration() error {
	failurePolicy := admissionv1.Ignore
	matchPolicy := admissionv1.Exact
	sideEffects := admissionv1.SideEffectClassNoneOnDryRun
	svcPort := webhookPort
	scopeAll := admissionv1.AllScopes
	path := "/mutate-pods"

	webhooks := []admissionv1.MutatingWebhook{
		{
			Name: webhookPodMutatorName,
			ClientConfig: admissionv1.WebhookClientConfig{
				Service: &admissionv1.ServiceReference{
					Name:      WebhookServiceName,
					Namespace: s.stos.Spec.GetResourceNS(),
					Port:      &svcPort,
					Path:      &path,
				},
			},
			Rules: []admissionv1.RuleWithOperations{
				{
					Operations: []admissionv1.OperationType{admissionv1.Create},
					Rule: admissionv1.Rule{
						APIGroups:   []string{""},
						APIVersions: []string{"v1"},
						Resources:   []string{"pods"},
						Scope:       &scopeAll,
					},
				},
			},
			FailurePolicy:           &failurePolicy,
			MatchPolicy:             &matchPolicy,
			NamespaceSelector:       &metav1.LabelSelector{},
			SideEffects:             &sideEffects,
			AdmissionReviewVersions: []string{"v1"},
		},
	}
	labels := podLabelsForAPIManager(s.stos.Name)

	return s.k8sResourceManager.MutatingWebhookConfiguration(MutatingWebhookConfigName, labels, webhooks).Create()
}

// deleteWebhookConfiguration deletes the webhook service and configuration.
func (s Deployment) deleteWebhookConfiguration() error {
	if err := s.k8sResourceManager.MutatingWebhookConfiguration(MutatingWebhookConfigName, nil, nil).Delete(); err != nil {
		return err
	}
	return s.k8sResourceManager.Service(WebhookServiceName, s.stos.Spec.GetResourceNS(), nil, nil, nil).Delete()
}
