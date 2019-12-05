package util

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	"github.com/blang/semver"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/storageos/cluster-operator/internal/pkg/discovery"
	storageos "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

// Constants used in NFS server test utils.
const (
	nfsServerName   = "example-nfsserver"
	nfsResourceSize = "1Gi"
	defaultNS       = "default"
)

// NewNFSServer returns a NFSServer object, created using a given NFS server
// spec.
func NewNFSServer(namespace string, nfsServerSpec storageos.NFSServerSpec) *storageos.NFSServer {
	return &storageos.NFSServer{
		TypeMeta: metav1.TypeMeta{
			Kind:       "NFSServer",
			APIVersion: "storageos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      nfsServerName,
			Namespace: namespace,
		},
		Spec: nfsServerSpec,
	}
}

// DeployNFSServer creates a custom resource and checks if the NFS Server
// statefulset is deployed successfully.
func DeployNFSServer(t *testing.T, ctx *framework.TestCtx, nfsServer *storageos.NFSServer) error {
	f := framework.Global

	err := f.Client.Create(goctx.TODO(), nfsServer, &framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval})
	if err != nil {
		return err
	}

	// Minimum version for running the complete test.
	minVersion := semver.Version{
		Major: 1, Minor: 13, Patch: 0,
	}

	featureSupported, err := featureSupportAvailable(minVersion)
	if err != nil {
		return fmt.Errorf("failed to check platform support for NFS Server test: %v", err)
	}

	if featureSupported {
		// Wait for NFS Server StatefulSet to be ready.
		err = WaitForStatefulSet(t, f.KubeClient, nfsServer.Namespace, nfsServer.Name, RetryInterval, Timeout*2)
		if err != nil {
			return err
		}

		// Check the Service endpoints to be selected properly.
		nfsServiceEndpoints := &corev1.Endpoints{}
		namespacedName := types.NamespacedName{
			Name:      nfsServer.Name,
			Namespace: nfsServer.Namespace,
		}
		if err := f.Client.Get(goctx.TODO(), namespacedName, nfsServiceEndpoints); err != nil {
			return err
		}
		if len(nfsServiceEndpoints.Subsets) < 1 {
			return fmt.Errorf("NFS Server Service has no selected endpoints")
		}
	} else {
		// Wait for 10 seconds here because there's no wait for the StatefulSet
		// to be ready. This will provide time for the PVC to be provisioned.
		time.Sleep(10 * time.Second)
		// Since the feature is not supported, only check if the StatefulSet
		// is created.
		statefulset := &appsv1.StatefulSet{}
		namespacedName := types.NamespacedName{
			Name:      nfsServer.Name,
			Namespace: nfsServer.Namespace,
		}
		if f.Client.Get(goctx.TODO(), namespacedName, statefulset); err != nil {
			return err
		}
	}

	return nil
}

// NFSServerTest creates a new NFSServer resource and checks if the resource is
// created and ready.
func NFSServerTest(t *testing.T, ctx *framework.TestCtx) {
	f := framework.Global

	// Create a NFS server spec.
	nfsServerSpec := storageos.NFSServerSpec{
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceStorage: resource.MustParse(nfsResourceSize),
			},
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "key",
				Operator: corev1.TolerationOpEqual,
				Value:    "value",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		},
	}

	// Create a new NFS server. This creates the server resources and checks the
	// resources to be ready.
	testNFSServer := NewNFSServer(defaultNS, nfsServerSpec)
	err := DeployNFSServer(t, ctx, testNFSServer)
	if err != nil {
		t.Fatal(err)
	}

	// Check if a ServiceMonitor was created.
	// ServiceMonitor is only created when the ServiceMonitor CRD is known in
	// the cluster.
	serviceMonitorExists, err := hasServiceMonitor()
	if err != nil {
		t.Error("failed to check if ServiceMonitor exists", err)
	}
	if serviceMonitorExists {
		serviceMonitor := &monitoringv1.ServiceMonitor{}
		smNSName := types.NamespacedName{
			Name:      fmt.Sprintf("%s-%s", testNFSServer.Name, "metrics"),
			Namespace: defaultNS,
		}
		if err := f.Client.Get(goctx.TODO(), smNSName, serviceMonitor); err != nil {
			t.Error("failed to get NFS metrics ServiceMonitor", err)
		}
	}

	// Delete the NFS server.
	if err := f.Client.Delete(goctx.TODO(), testNFSServer); err != nil {
		t.Error("failed to delete NFS Server", err)
	}

	// Wait for NFS resources to be deleted automatically.
	time.Sleep(5 * time.Second)
}

// hasServiceMonitor checks if Prometheus Service Monitor CRD is registered in
// the cluster.
func hasServiceMonitor() (bool, error) {
	apiVersion := "monitoring.coreos.com/v1"
	kind := "ServiceMonitor"
	return discovery.HasResource(framework.Global.KubeConfig, apiVersion, kind)
}
