package main

import (
	"context"
	"os"
	"runtime"
	"strings"
	"time"

	sdk "github.com/operator-framework/operator-sdk/pkg/sdk"
	sdkVersion "github.com/operator-framework/operator-sdk/version"
	"github.com/storageos/storageoscluster-operator/pkg/controller"
	stub "github.com/storageos/storageoscluster-operator/pkg/stub"

	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"

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

	k8sVersion, err := getK8SVersion(kubeclient)
	if err != nil {
		logrus.Errorf("failed to get k8s version: %v", err)
		os.Exit(1)
	}
	logrus.Infof("k8s version: %s", k8sVersion)

	operatorClient := controller.OperatorClient{}
	clusterController := controller.NewClusterController(operatorClient, strings.TrimLeft(k8sVersion, "v"))
	sdk.Handle(stub.NewHandler(eventRecorder(kubeclient), clusterController))
	sdk.Run(context.TODO())
}

func getK8SVersion(client kubernetes.Interface) (string, error) {
	info, err := client.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return info.String(), nil
}

// eventRecorder creates and returns an EventRecorder which could be used to
// broadcast events for k8s objects.
func eventRecorder(kubeClient kubernetes.Interface) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: kubeClient.CoreV1().Events(""),
		},
	)
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		v1.EventSource{Component: "storageoscluster-operator"},
	)
	return recorder
}
