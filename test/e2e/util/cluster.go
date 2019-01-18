package util

import (
	goctx "context"
	"fmt"
	"testing"
	"time"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	"github.com/storageos/cluster-operator/pkg/apis"
	storageos "github.com/storageos/cluster-operator/pkg/apis/storageos/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// Time constants.
const (
	RetryInterval        = time.Second * 5
	Timeout              = time.Second * 60
	CleanupRetryInterval = time.Second * 1
	CleanupTimeout       = time.Second * 5
)

// NewStorageOSCluster returns a StorageOSCluster object, created using a given
// cluster spec.
func NewStorageOSCluster(namespace string, clusterSpec storageos.StorageOSClusterSpec) *storageos.StorageOSCluster {
	return &storageos.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageOSCluster",
			APIVersion: "storageos.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "example-storageos",
			Namespace: namespace,
		},
		Spec: clusterSpec,
		Status: storageos.StorageOSClusterStatus{
			Nodes: []string{},
		},
	}
}

// SetupOperator installs the operator and ensures that the deployment is successful.
func SetupOperator(t *testing.T, ctx *framework.TestCtx) {
	clusterList := &storageos.StorageOSClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageOSCluster",
			APIVersion: "storageos.com/v1alpha1",
		},
	}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, clusterList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	err = ctx.InitializeClusterResources(&framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval})
	if err != nil {
		t.Fatalf("failed to initialize cluster resources: %v", err)
	}
	t.Log("Initialized cluster resources")

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatal(err)
	}

	f := framework.Global

	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "storageos-cluster-operator", 1, RetryInterval, Timeout)
	if err != nil {
		t.Fatal(err)
	}
}

// ClusterStatusCheck checks the values of cluster status based on a given
// number of nodes.
func ClusterStatusCheck(t *testing.T, status storageos.StorageOSClusterStatus, nodes int) {
	if len(status.Nodes) != nodes {
		t.Errorf("unexpected number of nodes:\n\t(GOT) %d\n\t(WNT) %d", len(status.Nodes), nodes)
	}

	if status.Phase != storageos.ClusterPhaseRunning {
		t.Errorf("unexpected cluster phase:\n\t(GOT) %s\n\t(WNT) %s", status.Phase, storageos.ClusterPhaseRunning)
	}

	wantReady := fmt.Sprintf("%d/%d", nodes, nodes)
	if status.Ready != wantReady {
		t.Errorf("unexpected Ready:\n\t(GOT) %s\n\t(WNT) %s", status.Ready, wantReady)
	}
}

// DeployCluster creates a custom resource and checks if the
// storageos daemonset is deployed successfully.
func DeployCluster(t *testing.T, ctx *framework.TestCtx, cluster *storageos.StorageOSCluster) error {
	f := framework.Global

	clusterSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-api",
			Namespace: "default",
		},
		Type: corev1.SecretType("kubernetes.io/storageos"),
		StringData: map[string]string{
			"apiUsername": "storageos",
			"apiPassword": "storageos",
		},
	}

	err := f.Client.Create(goctx.TODO(), clusterSecret, &framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval})
	if err != nil {
		t.Fatal(err)
	}

	err = f.Client.Create(goctx.TODO(), cluster, &framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval})
	if err != nil {
		return err
	}

	err = WaitForDaemonSet(t, f.KubeClient, "storageos", "storageos-daemonset", RetryInterval, Timeout*2)
	if err != nil {
		t.Fatal(err)
	}

	if cluster.Spec.CSI.Enable {
		err = WaitForStatefulSet(t, f.KubeClient, "storageos", "storageos-statefulset", RetryInterval, Timeout*2)
		if err != nil {
			t.Fatal(err)
		}
	}

	return nil
}

// WaitForDaemonSet checks and waits for a given daemonset to be in ready.
func WaitForDaemonSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		daemonset, err := kubeclient.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s daemonset\n", name)
				return false, nil
			}
			return false, err
		}

		if int(daemonset.Status.NumberReady) == 1 {
			return true, nil
		}

		t.Logf("Waiting for ready status of %s daemonset (%d)\n", name, daemonset.Status.NumberReady)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("DaemonSet Ready!\n")
	return nil
}

// WaitForStatefulSet checks and waits for a given statefulset to be in ready.
func WaitForStatefulSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		statefulset, err := kubeclient.AppsV1().StatefulSets(namespace).Get(name, metav1.GetOptions{IncludeUninitialized: true})
		if err != nil {
			if apierrors.IsNotFound(err) {
				t.Logf("Waiting for availability of %s statefulset\n", name)
				return false, nil
			}
			return false, err
		}

		if int(statefulset.Status.ReadyReplicas) == 1 {
			return true, nil
		}

		t.Logf("Waiting for ready status of %s statefulset (%d)\n", name, statefulset.Status.ReadyReplicas)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("StatefulSet Ready!\n")
	return nil
}
