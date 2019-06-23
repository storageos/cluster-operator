package main

import (
	"os"

	"github.com/storageos/cluster-operator/pkg/util/k8sutil"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("upgrader")

func main() {

	var log = logf.Log.WithName("upgrader")

	cfg, err := restclient.InClusterConfig()
	if err != nil {
		fatal(err)
	}

	client := kubernetes.NewForConfigOrDie(cfg)
	kops := k8sutil.NewK8SOps(client, log)

	newImage := os.Getenv("NEW_IMAGE")

	// Scale down the applications.
	if err = kops.ScaleDownApps(); err != nil {
		fatal(err)
	}

	// Update the storageos nodes.
	if err = kops.UpgradeDaemonSet(newImage); err != nil {
		fatal(err)
	}

	// Scale up the applications.
	if err = kops.ScaleUpApps(); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	log.Error(err, "Fatal error")
	os.Exit(1)
}
