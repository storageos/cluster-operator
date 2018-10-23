package main

import (
	"context"
	"os"
	"runtime"
	"strings"
	"time"

	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/storageos/cluster-operator/pkg/controller"
	stub "github.com/storageos/cluster-operator/pkg/stub"
	"github.com/storageos/cluster-operator/pkg/util/k8sutil"

	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/sirupsen/logrus"
)

func printVersion() {
	logrus.Infof("Go Version: %s", runtime.Version())
	logrus.Infof("Go OS/Arch: %s/%s", runtime.GOOS, runtime.GOARCH)
	logrus.Infof("operator-sdk Version: %v", sdkVersion.Version)
}

func main() {
	printVersion()

	sdk.ExposeMetricsPort()

	resource := "storageos.com/v1alpha1"
	kind := "StorageOSCluster"
	// Empty namespace to watch all the namespaces for the custom resource.
	namespace := ""
	resyncPeriod := time.Duration(10) * time.Second
	logrus.Infof("Watching %s, %s, %s, %d", resource, kind, namespace, resyncPeriod)
	sdk.Watch(resource, kind, namespace, resyncPeriod)
	kubeclient := k8sclient.GetKubeClient()

	k8sVersion, err := k8sutil.GetK8SVersion(kubeclient)
	if err != nil {
		logrus.Errorf("failed to get k8s version: %v", err)
		os.Exit(1)
	}
	logrus.Infof("k8s version: %s", k8sVersion)

	operatorClient := controller.OperatorClient{}
	clusterController := controller.NewClusterController(operatorClient, strings.TrimLeft(k8sVersion, "v"))
	sdk.Handle(stub.NewHandler(k8sutil.EventRecorder(kubeclient), clusterController))
	sdk.Run(context.TODO())
}
