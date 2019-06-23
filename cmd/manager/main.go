package main

import (
	"flag"
	"os"
	"runtime"

	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/storageos/cluster-operator/pkg/apis"
	"github.com/storageos/cluster-operator/pkg/controller"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

var log = logf.Log.WithName("setup")

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

	log.Info("Starting the Cmd")

	// Start the Cmd
	fatal(mgr.Start(signals.SetupSignalHandler()))
}

func fatal(err error) {
	log.Error(err, "Fatal error")
	os.Exit(1)
}
