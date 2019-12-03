package admission

import (
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/storageos/cluster-operator/internal/pkg/crv01/webhook"
	"github.com/storageos/cluster-operator/internal/pkg/crv01/webhook/admission"
	"github.com/storageos/cluster-operator/internal/pkg/crv01/webhook/admission/builder"
)

// MutatingWebhook is a mutating admission webhook.
type MutatingWebhook struct {
	// Name of the mutating webhook.
	Name string
	// Namespace where the mutating webhook is deployed.
	Namespace string
	// ServerName is the name of the admission webhook server.
	ServerName string
	// ServiceName is the Service for the webhook endpoints.
	ServiceName string
	// ServiceSelector is the label selector used by Service to select the pods.
	ServiceSelector map[string]string
	// Port is the port at which the webhook server runs.
	Port int32
	// Operations are the admission operation events to listen for.
	Operations []admissionregistrationv1beta1.OperationType
	// Manager is a controller manager.
	Manager manager.Manager
	// ObjectType is the type of the k8s object to mutate.
	ObjectType runtime.Object
	// Handlers are the webhook handlers.
	Handlers []admission.Handler
	// DisableConfigInstaller can be used to disable automated installation and
	// update of the mutating webhook configuration.
	DisableConfigInstaller *bool
}

// SetupWebhook setsup the webhook by building a mutating webhook, creating a
// server at a given port, a service, mutating webhook configuration and
// registering the created webhook with the created server.
func (w MutatingWebhook) SetupWebhook() error {
	mutatingWebhook, err := builder.NewWebhookBuilder().
		Name(w.Name).
		Mutating().
		Operations(w.Operations...).
		WithManager(w.Manager).
		ForType(w.ObjectType).
		Handlers(w.Handlers...).
		Build()
	if err != nil {
		return err
	}

	// Create a new server with the controller manager. Set the webhook
	// bootstrap to include a Service configuration.
	// NOTE: If we need to store the certificate outside of the operator, a
	// secret can be configured in the bootstrap. This way, the cert will be
	// generated just once and reused or shared by the operator(s).
	as, err := webhook.NewServer(w.ServerName, w.Manager, webhook.ServerOptions{
		Port:                          w.Port,
		DisableWebhookConfigInstaller: w.DisableConfigInstaller,
		BootstrapOptions: &webhook.BootstrapOptions{
			Service: &webhook.Service{
				Name:      w.ServiceName,
				Namespace: w.Namespace,
				Selectors: w.ServiceSelector,
			},
			MutatingWebhookConfigName: w.ServerName,
		},
	})
	if err != nil {
		return err
	}

	// Register the webhook with the server.
	if err := as.Register(mutatingWebhook); err != nil {
		return err
	}

	return nil
}
