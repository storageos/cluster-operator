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

	// WebhookServiceFor is the webhook service's value for the
	// app.kubernetes.io/service-for label.
	WebhookServiceFor = "storageos-webhook-server"

	// ServicePort configuration.
	webhookPortName         = "webhooks"
	webhookPort       int32 = 443
	webhookTargetPort int32 = 9443

	mutatePodPath = "/mutate-pods"
	mutatePVCPath = "/mutate-pvcs"

	// webhookPodMutatorName and webhookPVCMutatorName are the names of the
	// webhooks.
	webhookPodMutatorName = "pod-mutator.storageos.com"
	webhookPVCMutatorName = "pvc-mutator.storageos.com"
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
	labels := make(map[string]string)
	for k, v := range podLabelsForAPIManager(s.stos.Name) {
		labels[k] = v
	}
	labels[k8s.ServiceFor] = WebhookServiceFor

	return s.k8sResourceManager.Service(WebhookServiceName, s.stos.Spec.GetResourceNS(), labels, nil, spec).Create()
}

// createMutatingWebhookConfiguration creates the configuration for the mutating
// webhooks.  The configuration will be updated by the api-manager to include
// the CA cert bundle.
func (s Deployment) createMutatingWebhookConfiguration() error {
	scopeAll := admissionv1.AllScopes
	webhooks := []admissionv1.MutatingWebhook{
		s.mutatingWebhookConfiguration(webhookPodMutatorName, mutatePodPath, admissionv1.Ignore, []admissionv1.RuleWithOperations{
			{
				Operations: []admissionv1.OperationType{admissionv1.Create},
				Rule: admissionv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"pods"},
					Scope:       &scopeAll,
				},
			},
		}),
		s.mutatingWebhookConfiguration(webhookPVCMutatorName, mutatePVCPath, admissionv1.Ignore, []admissionv1.RuleWithOperations{
			{
				Operations: []admissionv1.OperationType{admissionv1.Create},
				Rule: admissionv1.Rule{
					APIGroups:   []string{""},
					APIVersions: []string{"v1"},
					Resources:   []string{"persistentvolumeclaims"},
					Scope:       &scopeAll,
				},
			},
		}),
	}
	labels := podLabelsForAPIManager(s.stos.Name)

	return s.k8sResourceManager.MutatingWebhookConfiguration(MutatingWebhookConfigName, labels, webhooks).Create()
}

func (s Deployment) mutatingWebhookConfiguration(name string, path string, failurePolicy admissionv1.FailurePolicyType, rules []admissionv1.RuleWithOperations) admissionv1.MutatingWebhook {
	matchPolicy := admissionv1.Equivalent
	sideEffects := admissionv1.SideEffectClassNoneOnDryRun
	port := webhookPort

	return admissionv1.MutatingWebhook{
		Name: name,
		ClientConfig: admissionv1.WebhookClientConfig{
			Service: &admissionv1.ServiceReference{
				Name:      WebhookServiceName,
				Namespace: s.stos.Spec.GetResourceNS(),
				Port:      &port,
				Path:      &path,
			},
		},
		Rules:                   rules,
		FailurePolicy:           &failurePolicy,
		MatchPolicy:             &matchPolicy,
		NamespaceSelector:       &metav1.LabelSelector{},
		SideEffects:             &sideEffects,
		AdmissionReviewVersions: []string{"v1"},
	}
}

// deleteWebhookConfiguration deletes the webhook service and configuration.
func (s Deployment) deleteWebhookConfiguration() error {
	if err := s.k8sResourceManager.MutatingWebhookConfiguration(MutatingWebhookConfigName, nil, nil).Delete(); err != nil {
		return err
	}
	return s.k8sResourceManager.Service(WebhookServiceName, s.stos.Spec.GetResourceNS(), nil, nil, nil).Delete()
}
