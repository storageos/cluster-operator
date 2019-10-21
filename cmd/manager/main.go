package main

import (
	"flag"
	"os"
	"runtime"
	"strconv"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/storageos/cluster-operator/internal/pkg/admission"
	"github.com/storageos/cluster-operator/internal/pkg/admission/scheduler"
	"github.com/storageos/cluster-operator/pkg/apis"
	"github.com/storageos/cluster-operator/pkg/controller"
	"github.com/storageos/cluster-operator/pkg/storageos"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
	webhookAdmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

var log = logf.Log.WithName("storageos.setup")

const (
	// Env vars.
	enableSchedulerEnvVar = "ENABLE_SCHEDULER"
	podNamespaceEnvVar    = "POD_NAMESPACE"

	// operatorNameLabel is the "name" label in the operator's deployment
	// config. This is needed for proper operator pod selection by webhook
	// service.
	// NOTE: If the operator's deployment changes the name label, this must be
	// updated.
	operatorNameLabel = "storageos-cluster-operator"

	// podSchedulerWebhookName is a fully qualified name of the pod scheduler
	// admission webhook.
	podSchedulerWebhookName = "podscheduler.storageos.com"
	// podSchedulerResourceName is the name of webhook server, service,
	// mutatingwebhookconfig and other related resources.
	podSchedulerResourceName = "storageos-scheduler-webhook"
	// podSchedulerWebhookPort is the port at which the pod scheduler webhook
	// server runs.
	podSchedulerWebhookPort = 5720
)

func main() {

	logf.SetLogger(logf.ZapLogger(true))

	log.WithValues(
		"goversion", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"operator-sdk", sdkVersion.Version,
	).Info("Initializing")

	flag.Parse()

	// TODO: Expose metrics port after SDK uses controller-runtime's dynamic client
	// sdk.ExposeMetricsPort()

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	// Use "" namespace to watch all the namespaces.
	mgr, err := manager.New(cfg, manager.Options{Namespace: ""})
	if err != nil {
		fatal(err)
	}

	log.Info("Registering Components")

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		fatal(err)
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		fatal(err)
	}

	// Check if storageos scheduler should be enabled.
	enableScheduler := false
	enableSchedulerEnvVarVal := os.Getenv(enableSchedulerEnvVar)
	if len(enableSchedulerEnvVarVal) > 0 {
		var parseError error
		enableScheduler, parseError = strconv.ParseBool(enableSchedulerEnvVarVal)
		if parseError != nil {
			log.Error(parseError, "unable to parse ENABLE_SCHEDULER val")
			fatal(parseError)
		}
	}

	if enableScheduler {
		// Configure a pod scheduler webhook handler with StorageOS provisioner
		// and scheduler.
		webhookHandler := &scheduler.PodSchedulerSetter{
			Provisioners:  []string{storageos.CSIProvisionerName, storageos.IntreeProvisionerName},
			SchedulerName: storageos.SchedulerExtenderName,
		}

		// Enable webhook config installer.
		disableWebhookConfigInstaller := false

		// Create a mutating webhook for mutating pods that have volumes managed
		// by StorageOS and set them to use storageos pod scheduler.
		mutatingWebhook := &admission.MutatingWebhook{
			Name:        podSchedulerWebhookName,
			Namespace:   os.Getenv(podNamespaceEnvVar),
			ServerName:  podSchedulerResourceName,
			ServiceName: podSchedulerResourceName,
			ServiceSelector: map[string]string{
				"name": operatorNameLabel,
			},
			Port: podSchedulerWebhookPort,
			Operations: []admissionregistrationv1beta1.OperationType{
				admissionregistrationv1beta1.Create,
			},
			Manager:                mgr,
			ObjectType:             &corev1.Pod{},
			Handlers:               []webhookAdmission.Handler{webhookHandler},
			DisableConfigInstaller: &disableWebhookConfigInstaller,
		}

		if err := mutatingWebhook.SetupWebhook(); err != nil {
			fatal(err)
		}
	}

	log.Info("Starting the StorageOS Operator")

	// Start the Cmd
	fatal(mgr.Start(signals.SetupSignalHandler()))
}

func fatal(err error) {
	log.Error(err, "Fatal error")
	os.Exit(1)
}
