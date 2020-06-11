package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
	"strconv"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	admissionregistrationv1beta1 "k8s.io/api/admissionregistration/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/storageos/cluster-operator/internal/pkg/admission"
	"github.com/storageos/cluster-operator/internal/pkg/admission/scheduler"
	webhookAdmission "github.com/storageos/cluster-operator/internal/pkg/crv01/webhook/admission"
	"github.com/storageos/cluster-operator/pkg/apis"
	"github.com/storageos/cluster-operator/pkg/controller"
	"github.com/storageos/cluster-operator/pkg/storageos"
)

var log = logf.Log.WithName("storageos.setup")

const (
	// Env vars.
	disableSchedulerWebhookEnvVar = "DISABLE_SCHEDULER_WEBHOOK"
	podNamespaceEnvVar            = "POD_NAMESPACE"

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
	// podSchedulerAnnotationKey is the pod annotation key that can be set to
	// skip pod scheduler name mutation.
	podSchedulerAnnotationKey = "storageos.com/scheduler"
)

var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

func main() {

	logf.SetLogger(zap.Logger(true))

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

	ctx := context.TODO()
	// Become the leader before proceeding.
	err = leader.Become(ctx, "storageos-operator-lock")
	if err != nil {
		fatal(err)
	}

	// Create a new Cmd to provide shared dependencies and start components
	// Use "" namespace to watch all the namespaces.
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          "",
		MetricsBindAddress: fmt.Sprintf("%s:%d", metricsHost, metricsPort),
	})
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

	// Check if storageos pod scheduler webhook should be disabled.
	disableSchedulerWebhook := false
	disableSchedulerEnvVarVal := os.Getenv(disableSchedulerWebhookEnvVar)
	if len(disableSchedulerEnvVarVal) > 0 {
		var parseError error
		disableSchedulerWebhook, parseError = strconv.ParseBool(disableSchedulerEnvVarVal)
		if parseError != nil {
			log.Error(parseError, "unable to parse ENABLE_SCHEDULER val")
			fatal(parseError)
		}
	}

	if !disableSchedulerWebhook {
		// Configure a pod scheduler webhook handler with StorageOS provisioner
		// and scheduler.
		webhookHandler := &scheduler.PodSchedulerSetter{
			Provisioners:           []string{storageos.CSIProvisionerName, storageos.IntreeProvisionerName, storageos.StorageOSProvisionerName},
			SchedulerName:          storageos.SchedulerExtenderName,
			SchedulerAnnotationKey: podSchedulerAnnotationKey,
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

	if err := serveCRMetrics(cfg); err != nil {
		log.Info("Could not generate and serve custom resource metrics", "error", err.Error())
	}

	// Add to the below struct any other metrics ports to expose.
	servicePorts := []corev1.ServicePort{
		{Port: metricsPort, Name: metrics.OperatorPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: metricsPort}},
		{Port: operatorMetricsPort, Name: metrics.CRPortName, Protocol: corev1.ProtocolTCP, TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: operatorMetricsPort}},
	}

	// CreateServiceMonitors will automatically create the prometheus-operator
	// ServiceMonitor resources necessary to configure Prometheus to scrape
	// metrics from this operator.
	services := []*corev1.Service{}

	// Create Service object to expose the metrics port(s).
	service, err := metrics.CreateMetricsService(ctx, cfg, servicePorts)
	if err != nil {
		log.Info("Could not create metrics Service", "error", err.Error())
	} else {
		services = append(services, service)
	}

	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		log.Info("Could not get the operator namespace, not creating ServiceMonitor", "error", err.Error())
	} else {
		_, err = metrics.CreateServiceMonitors(cfg, operatorNs, services)
		if err != nil {
			log.Info("Could not create ServiceMonitor object", "error", err.Error())
			// If this operator is deployed to a cluster without the
			// prometheus-operator running, it will return
			// ErrServiceMonitorNotPresent, which can be used to safely skip ServiceMonitor creation.
			if err == metrics.ErrServiceMonitorNotPresent {
				log.Info("Install prometheus-operator in your cluster to create ServiceMonitor objects", "error", err.Error())
			}
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

// serveCRMetrics gets the Operator/CustomResource GVKs and generates metrics based on those types.
// It serves those metrics on "http://metricsHost:operatorMetricsPort".
func serveCRMetrics(cfg *rest.Config) error {
	// Below function returns filtered operator/CustomResource specific GVKs.
	// For more control override the below GVK list with your own custom logic.
	filteredGVK, err := k8sutil.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := k8sutil.GetOperatorNamespace()
	if err != nil {
		return err
	}
	// To generate metrics in other namespaces, add the values below.
	ns := []string{operatorNs}
	// Generate and serve custom resource specific metrics.
	err = kubemetrics.GenerateAndServeCRMetrics(cfg, ns, filteredGVK, metricsHost, operatorMetricsPort)
	if err != nil {
		return err
	}
	return nil
}
