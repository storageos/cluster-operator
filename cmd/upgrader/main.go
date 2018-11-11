package main

import (
	"log"
	"os"

	"github.com/storageos/cluster-operator/pkg/util/k8sutil"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
)

func main() {
	cfg, err := restclient.InClusterConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := kubernetes.NewForConfigOrDie(cfg)
	kops := k8sutil.NewK8SOps(client)

	newImage := os.Getenv("NEW_IMAGE")

	// Scale down the applications.
	if err = kops.ScaleDownApps(); err != nil {
		log.Fatal(err)
	}

	// Update the storageos nodes.
	if err = kops.UpgradeDaemonSet(newImage); err != nil {
		log.Fatal(err)
	}

	// Scale up the applications.
	if err = kops.ScaleUpApps(); err != nil {
		log.Fatal(err)
	}
}
