package util

import (
	goctx "context"
	"testing"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	storageosapi "github.com/storageos/cluster-operator/internal/pkg/storageos"
	storageos "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

// StorageOSClusterUpdateTest fetches a StorageOSCluster CR object, updates it, and
// checks if the changes have propogated correctly.
func StorageOSClusterUpdateTest(t *testing.T, ctx *framework.Context) {
	f := framework.Global

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("failed to get operator namespace: %v", err)
	}
	nsName := types.NamespacedName{
		Name:      TestClusterCRName,
		Namespace: namespace,
	}

	stos := &storageos.StorageOSCluster{}
	if err := f.Client.Get(goctx.TODO(), nsName, stos); err != nil {
		t.Fatalf("failed to get storageoscluster: %v", err)
	}

	// Double-check defaults were set as we'd expect.
	if stos.Spec.Debug {
		t.Errorf("spec.Debug expected false")
	}
	if stos.Spec.DisableTelemetry {
		t.Errorf("spec.DisableTelemetry expected false")
	}

	// Change the cluster config.  Only change things that aren't going to break
	// the cluster.  Most will need a cluster restart to apply anyways.
	stos.Spec.Debug = true
	stos.Spec.DisableTelemetry = true

	if err := f.Client.Update(goctx.TODO(), stos); err != nil {
		t.Errorf("failed to update storageoscluster: %v", err)
	}

	// Allow 10s for operator to make changes.  I'd rather use a wait, but this
	// should be good for the e2e test.
	time.Sleep(10 * time.Second)

	// Verify the update has changed the ConfigMap.
	config := &corev1.ConfigMap{}
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "storageos-node-config", Namespace: stos.Spec.GetResourceNS()}, config); err != nil {
		t.Fatalf("failed to get configmap: %v", err)
	}
	if config.Data == nil {
		t.Fatal("ConfigMap data nil")
	}

	if config.Data["LOG_LEVEL"] != "debug" {
		t.Errorf("ConfigMap expected LOG_LEVEL = %q, got %q", "debug", config.Data["LOG_LEVEL"])
	}
	if config.Data["DISABLE_TELEMETRY"] != "true" {
		t.Errorf("ConfigMap expected DISABLE_TELEMETRY = %q, got %q", "true", config.Data["DISABLE_TELEMETRY"])
	}

	if t.Failed() {
		logs, logErr := GetOperatorLogs(namespace)
		if logErr != nil {
			t.Error(errors.Wrap(logErr, "failed to fetch operator logs"))
		}
		t.Log(logs)
	}

	// Verify the update has changed the control plane's cluster config.
	//
	// Since the e2e test doesn't run in the cluster, go direct to the host ip.
	var client *storageosapi.Client
	var clientErr error
	for _, ip := range stos.Status.Members.Ready {
		client = storageosapi.New(ip)
		clientErr = client.Authenticate(apiUsername, apiPassword)
		if clientErr == nil {
			break
		}
	}
	if clientErr != nil || client == nil {
		t.Fatalf("failed to get storageos api client: %v", err)
	}
	cluster, err := client.GetCluster(goctx.Background())
	if err != nil {
		t.Fatalf("failed to get cluster from api: %v", err)
	}

	if cluster.LogLevel != "debug" {
		t.Errorf("api cluster expected LogLevel = %q, got %q", "debug", cluster.LogLevel)
	}
	if cluster.DisableTelemetry != stos.Spec.DisableTelemetry {
		t.Errorf("api cluster expected DisableTelemetry = %t, got %t", stos.Spec.DisableTelemetry, cluster.DisableTelemetry)
	}
	// CrashReporting and VersionCheck both use value of Telemetry.
	if cluster.DisableCrashReporting != stos.Spec.DisableTelemetry {
		t.Errorf("api cluster expected DisableCrashReporting = %t, got %t", stos.Spec.DisableTelemetry, cluster.DisableCrashReporting)
	}
	if cluster.DisableVersionCheck != stos.Spec.DisableTelemetry {
		t.Errorf("api cluster expected DisableVersionCheck = %t, got %t", stos.Spec.DisableTelemetry, cluster.DisableVersionCheck)
	}
}
