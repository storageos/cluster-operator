package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	oputils "github.com/operator-framework/operator-sdk/pkg/k8sutil"
	kubemetrics "github.com/operator-framework/operator-sdk/pkg/kube-metrics"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/metrics"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	"github.com/storageos/cluster-operator/pkg/apis"
	"github.com/storageos/cluster-operator/pkg/controller"
	"github.com/storageos/cluster-operator/pkg/util"
	"github.com/storageos/cluster-operator/pkg/util/k8sutil"
)

const supportedMinVersion = "1.17.0"

var log = logf.Log.WithName("storageos.setup")

var (
	metricsHost               = "0.0.0.0"
	metricsPort         int32 = 8383
	operatorMetricsPort int32 = 8686
)

func main() {
	logf.SetLogger(zap.New(zap.UseDevMode(true)))

	log.WithValues(
		"goversion", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"operator-sdk", sdkVersion.Version,
	).Info("Initializing")

	flag.Parse()

	// TODO: Expose metrics port after SDK uses controller-runtime's dynamic client
	// sdk.ExposeMetricsPort()

	// Get a config to talk to the apiserver.
	cfg, err := config.GetConfig()
	if err != nil {
		fatal(err)
	}

	// Validate Kubernetes version
	if supported, err := versionSupported(cfg); err != nil {
		fatal(err)
	} else if !supported {
		fatal(fmt.Errorf("current version of Kubernetes is lower than required minimum version [%s]", supportedMinVersion))
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

	// Setup Scheme for all resources.
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		fatal(err)
	}

	// Required from ServiceMonitor management
	if err := monitoringv1.AddToScheme(mgr.GetScheme()); err != nil {
		log.Info(fmt.Sprintf("failed to register monitoring api for managing prometheus service monitors: %s", err))
	}

	// Setup all Controllers
	if err := controller.AddToManager(mgr); err != nil {
		fatal(err)
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

	operatorNs, err := oputils.GetOperatorNamespace()
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
	filteredGVK, err := oputils.GetGVKsFromAddToScheme(apis.AddToScheme)
	if err != nil {
		return err
	}
	// Get the namespace the operator is currently deployed in.
	operatorNs, err := oputils.GetOperatorNamespace()
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

func versionSupported(config *rest.Config) (bool, error) {
	client := kubernetes.NewForConfigOrDie(config)
	kops := k8sutil.NewK8SOps(client, log)
	version, err := kops.GetK8SVersion()
	if err != nil {
		return false, err
	}

	return util.VersionSupported(version, supportedMinVersion), nil
}
