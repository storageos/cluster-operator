package util

import (
	"context"
	goctx "context"
	"fmt"
	"os/exec"
	"testing"
	"time"

	"github.com/blang/semver"
	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/operator-framework/operator-sdk/pkg/test/e2eutil"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1beta1 "k8s.io/api/storage/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/storageos/cluster-operator/pkg/apis"
	storageos "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	deploy "github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/pkg/util/k8s"
	"github.com/storageos/cluster-operator/pkg/util/k8sutil"
)

// Time constants.
const (
	RetryInterval        = time.Second * 5
	Timeout              = time.Second * 90
	CleanupRetryInterval = time.Second * 1
	CleanupTimeout       = time.Second * 15
)

// TestClusterCRName is the name of StorageOSCluster CR used in the tests.
const TestClusterCRName string = "example-storageos"

// NewStorageOSCluster returns a StorageOSCluster object, created using a given
// cluster spec.
func NewStorageOSCluster(namespace string, clusterSpec storageos.StorageOSClusterSpec) *storageos.StorageOSCluster {
	return &storageos.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageOSCluster",
			APIVersion: "storageos.com/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      TestClusterCRName,
			Namespace: namespace,
		},
		Spec: clusterSpec,
		Status: storageos.StorageOSClusterStatus{
			Nodes: []string{},
		},
	}
}

// SetupOperator installs the operator and ensures that the deployment is successful.
func SetupOperator(t *testing.T, ctx *framework.Context) {
	clusterList := &storageos.StorageOSClusterList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StorageOSCluster",
			APIVersion: "storageos.com/v1",
		},
	}
	err := framework.AddToFrameworkScheme(apis.AddToScheme, clusterList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme to framework: %v", err)
	}

	// Add ServiceMonitor Scheme to framework's scheme to be used with the
	// dynamic client.
	serviceMonitorList := &monitoringv1.ServiceMonitorList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceMonitor",
			APIVersion: "monitoring.coreos.com/v1",
		},
	}
	err = framework.AddToFrameworkScheme(monitoringv1.AddToScheme, serviceMonitorList)
	if err != nil {
		t.Fatalf("failed to add custom resource scheme ServiceMonitor to framework: %v", err)
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

	// Create webhook resources to test migration. These resources should be
	// deleted by the operator at startup.
	oldWebhookResourceName := "storageos-scheduler-webhook"

	whc := &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: oldWebhookResourceName,
		},
	}
	if err := f.Client.Create(context.TODO(), whc, nil); err != nil {
		t.Errorf("failed to create webhook config: %v", err)
	}

	// Valid spec.ports is required to create a real service.
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oldWebhookResourceName,
			Namespace: namespace,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "foo",
					Protocol:   "TCP",
					Port:       int32(666),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(777)},
				},
			},
		},
	}
	if err := f.Client.Create(context.TODO(), svc, nil); err != nil {
		t.Errorf("failed to create webhook service: %v", err)
	}

	// Deploy the operator.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, namespace, "storageos-cluster-operator", 1, RetryInterval, Timeout)
	if err != nil {
		t.Fatal(err)
	}

	// Check if the operator delete the webhook resources.
	whcKey, keyErr := client.ObjectKeyFromObject(whc)
	if keyErr != nil {
		t.Errorf("failed to get object key from wehbook config: %v", keyErr)
	}
	if getErr := f.Client.Get(context.TODO(), whcKey, whc); getErr == nil {
		t.Error("webhook configuration still exists after running migration")
	}

	svcKey, keyErr := client.ObjectKeyFromObject(svc)
	if keyErr != nil {
		t.Errorf("failed to get object key from wehbook service: %v", keyErr)
	}
	if getErr := f.Client.Get(context.TODO(), svcKey, svc); getErr == nil {
		t.Error("webhook service still exists after running migration")
	}
}

// ClusterStatusCheck checks the values of cluster status based on a given
// number of nodes.
func ClusterStatusCheck(t *testing.T, nsName types.NamespacedName, nodes int, retryInterval, timeout time.Duration) error {
	f := framework.Global

	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		cluster := storageos.StorageOSCluster{}
		err = f.Client.Get(context.TODO(), nsName, &cluster)
		if err != nil {
			t.Logf("failed to get the cluster: %v, retrying...", err)
			return false, nil
		}
		status := cluster.Status

		if len(status.Nodes) != nodes {
			t.Logf("unexpected number of nodes:\n\t(GOT) %d\n\t(WNT) %d, retrying...", len(status.Nodes), nodes)
			return false, nil
		}

		if status.Phase != storageos.ClusterPhaseRunning {
			t.Logf("unexpected cluster phase:\n\t(GOT) %s\n\t(WNT) %s, retrying...", status.Phase, storageos.ClusterPhaseRunning)
			return false, nil
		}

		wantReady := fmt.Sprintf("%d/%d", nodes, nodes)
		if status.Ready != wantReady {
			t.Logf("unexpected Ready:\n\t(GOT) %s\n\t(WNT) %s, retrying...", status.Ready, wantReady)
			return false, nil
		}

		if len(status.Members.Ready) != nodes {
			t.Logf("unexpected number of ready members:\n\t(GOT) %d\n\t(WNT) %d, retrying...", len(status.Members.Ready), nodes)
			return false, nil
		}

		if len(status.Members.Unready) != 0 {
			t.Logf("unexpected number of unready members:\n\t(GOT) %d\n\t(WNT) %d, retrying...", len(status.Members.Unready), 0)
			return false, nil
		}

		return true, nil
	})
	if err != nil {
		return fmt.Errorf("cluster status check failed: %v", err)
	}
	t.Logf("Cluster Status Ready!\n")
	return nil
}

// DeployCluster creates a custom resource and checks if the
// storageos daemonset is deployed successfully.
func DeployCluster(t *testing.T, ctx *framework.Context, cluster *storageos.StorageOSCluster) error {
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

	err = WaitForDaemonSet(t, f.KubeClient, cluster.Spec.GetResourceNS(), "storageos-daemonset", RetryInterval, Timeout*2)
	if err != nil {
		t.Fatal(err)
	}

	// Wait for the CSI helpers.
	err = e2eutil.WaitForDeployment(t, f.KubeClient, cluster.Spec.GetResourceNS(), "storageos-csi-helper", 1, RetryInterval, Timeout*2)
	if err != nil {
		t.Fatal(err)
	}

	return nil
}

// WaitForDaemonSet checks and waits for a given daemonset to be in ready.
func WaitForDaemonSet(t *testing.T, kubeclient kubernetes.Interface, namespace, name string, retryInterval, timeout time.Duration) error {
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		daemonset, err := kubeclient.AppsV1().DaemonSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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
		statefulset, err := kubeclient.AppsV1().StatefulSets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
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

// NodeLabelSyncTest adds a new label to k8s node and checks if the label is
// synced to the storageos node labels.
func NodeLabelSyncTest(t *testing.T, kubeclient kubernetes.Interface) {
	// Test labels that are added on k8s node.
	testLabelKey := "foo10"
	testLabelVal := "bar10"

	// Script that queries storageos node info and checks if the labels are set.
	nodeLabelCheckScript := "./test/port-forward.sh"
	expectedScriptOutput := "0\n"

	// Get the existing node to update its labels.
	nodes, err := kubeclient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		t.Fatalf("failed to get nodes: %v", err)
	}
	node := nodes.Items[0]
	labels := node.GetLabels()
	labels[testLabelKey] = testLabelVal
	node.SetLabels(labels)
	_, err = kubeclient.CoreV1().Nodes().Update(context.TODO(), &node, metav1.UpdateOptions{})
	if err != nil {
		t.Errorf("failed to update node labels: %v", err)
		return
	}

	// Cleanup - remove the label from k8s node at the end.
	defer func() {
		// Get the latest version of node to update.
		nodes, err = kubeclient.CoreV1().Nodes().List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			t.Errorf("failed to get nodes: %v", err)
			return
		}
		node = nodes.Items[0]
		labels = node.GetLabels()
		delete(labels, testLabelKey)
		node.SetLabels(labels)
		_, err = kubeclient.CoreV1().Nodes().Update(context.TODO(), &node, metav1.UpdateOptions{})
		if err != nil {
			t.Errorf("failed to cleanup node labels: %v", err)
			return
		}
	}()

	// Wait for the node-controller to update storageos node.
	time.Sleep(5 * time.Second)

	out, err := exec.Command(nodeLabelCheckScript).Output()
	if err != nil {
		t.Errorf("failed while executing script: %v", err)
		return
	}
	if string(out) != expectedScriptOutput {
		t.Errorf("unexpected script output: %v", string(out))
		return
	}
}

// StorageOSClusterCRAttributesTest fetches a StorageOSCluster CR object and
// checks if the CR properties are unset.
func StorageOSClusterCRAttributesTest(t *testing.T, crName string, crNamespace string) {
	f := framework.Global

	testStorageOS := &storageos.StorageOSCluster{}
	err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: crName, Namespace: crNamespace}, testStorageOS)
	if err != nil {
		t.Error(err)
	}

	// Check if the CR has the defaults and inferred attributes in the spec.
	if testStorageOS.Spec.Join == "" {
		t.Errorf("spec.join must not be empty")
	}

	if testStorageOS.Spec.Namespace == "" {
		t.Errorf("spec.namespace must not be empty")
	}

	if testStorageOS.Spec.Service.Name == "" {
		t.Errorf("spec.service.name must not be empty")
	}
}

// featureSupportAvailable can be used by tests to check if the platform
// supports the test by passing a minimum version of k8s required to run the
// test.
func featureSupportAvailable(minVersion semver.Version) (bool, error) {
	log := logf.Log.WithName("test.featureSupportAvailability")
	k := k8sutil.NewK8SOps(framework.Global.KubeClient, log)
	version, err := k.GetK8SVersion()
	if err != nil {
		return false, fmt.Errorf("failed to get k8s version: %v", err)
	}

	currentVersion, err := semver.Parse(version)
	if err != nil {
		return false, fmt.Errorf("failed to parse k8s version: %v", err)
	}

	if currentVersion.Compare(minVersion) >= 0 {
		// This test is supported in this version of k8s.
		return true, nil
	}

	// Test is not supported in this version of k8s. Skip the test.
	return false, nil
}

// CSIDriverResourceTest checks if the CSIDriver resource is created. In k8s
// 1.14+, CSIDriver is created as part of the cluster deployment.
func CSIDriverResourceTest(t *testing.T, driverName string) {
	k8sVerMajor := 1
	k8sVerMinor := 14
	k8sVerPatch := 0

	// Minimum version of k8s required to run this test.
	minVersion := semver.Version{
		Major: uint64(k8sVerMajor),
		Minor: uint64(k8sVerMinor),
		Patch: uint64(k8sVerPatch),
	}

	// Check the k8s version before running this test. CSIDriver built-in
	// resource does not exists in openshift 3.11 (k8s 1.11).
	featureSupported, err := featureSupportAvailable(minVersion)
	if err != nil {
		t.Errorf("failed to check platform support for CSIDriver test: %v", err)
		return
	}

	// Skip if the feature is not supported.
	if !featureSupported {
		return
	}

	f := framework.Global
	csiDriver := &storagev1beta1.CSIDriver{}
	err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: driverName}, csiDriver)
	if err != nil {
		t.Errorf("CSIDriver not found: %v", err)
	}
}

// APIManagerDeploymentTest checks the api-manager deployment.
func APIManagerDeploymentTest(t *testing.T, ns string, retryInterval, timeout time.Duration) {
	f := framework.Global
	err := e2eutil.WaitForDeployment(t, f.KubeClient, ns, deploy.APIManagerName, 2, retryInterval, timeout)
	if err != nil {
		t.Fatalf("timed out waiting for api-manager deployment: err=%v", err)
	}

	var pods *corev1.PodList
	err = wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		pods, err = f.KubeClient.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", k8s.AppComponent, deploy.APIManagerName),
		})
		if err != nil {
			return false, err
		}
		if len(pods.Items) == 2 {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for api-manager pods: %v", err)
	}

	// check label needed for metrics service.
	for _, pod := range pods.Items {
		got, ok := pod.Labels[k8s.AppComponent]
		if !ok {
			t.Errorf("expected label %q not set", k8s.AppComponent)
		}
		if ok && got != deploy.APIManagerName {
			t.Errorf("expected label %q set to %q, want %q", k8s.AppComponent, got, deploy.APIManagerName)
		}
	}
}

// APIManagerWebhookServiceTest checks the api-manager webhook service deployment.
func APIManagerWebhookServiceTest(t *testing.T, ns string, retryInterval, timeout time.Duration) {
	f := framework.Global
	var svc *corev1.Service
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		svc, err = f.KubeClient.CoreV1().Services(ns).Get(context.TODO(), deploy.WebhookServiceName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if svc.Spec.ClusterIP != "" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for api-manager webhook service: %v", err)
	}

	// Check expected labels.
	got, ok := svc.Labels[k8s.AppComponent]
	if !ok {
		t.Errorf("expected label %q not set", k8s.AppComponent)
	}
	if ok && got != deploy.APIManagerName {
		t.Errorf("expected label %q set to %q, want %q", k8s.AppComponent, got, deploy.APIManagerName)
	}
	got, ok = svc.Labels[k8s.ServiceFor]
	if !ok {
		t.Errorf("expected label %q not set", k8s.ServiceFor)
	}
	if ok && got != deploy.WebhookServiceFor {
		t.Errorf("expected label %q set to %q, want %q", k8s.ServiceFor, got, deploy.WebhookServiceFor)
	}
}

// APIManagerMetricsServiceTest checks the api-manager metrics service deployment.
func APIManagerMetricsServiceTest(t *testing.T, ns string, retryInterval, timeout time.Duration) {
	f := framework.Global
	var svc *corev1.Service
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		svc, err = f.KubeClient.CoreV1().Services(ns).Get(context.TODO(), deploy.APIManagerMetricsName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if svc.Spec.ClusterIP != "" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for api-manager metrics service: %v", err)
	}

	// Check label needed for ServiceMonitor.
	got, ok := svc.Labels[k8s.AppComponent]
	if !ok {
		t.Errorf("expected label %q not set", k8s.AppComponent)
	}
	if ok && got != deploy.APIManagerName {
		t.Errorf("expected label %q set to %q, want %q", k8s.AppComponent, got, deploy.APIManagerName)
	}

	got, ok = svc.Labels[k8s.ServiceFor]
	if !ok {
		t.Errorf("expected label %q not set", k8s.ServiceFor)
	}
	if ok && got != deploy.APIManagerName {
		t.Errorf("expected label %q set to %q, want %q", k8s.ServiceFor, got, deploy.APIManagerName)
	}
}

// APIManagerMetricsServiceMonitorTest checks the api-manager metrics service monitor.
func APIManagerMetricsServiceMonitorTest(t *testing.T, ns string, retryInterval, timeout time.Duration) {
	f := framework.Global
	nn := types.NamespacedName{
		Name:      deploy.APIManagerMetricsName,
		Namespace: ns,
	}
	sm := &monitoringv1.ServiceMonitor{}
	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = f.Client.Get(context.TODO(), nn, sm)
		if err != nil {
			return false, err
		}
		if sm != nil {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		t.Fatalf("timed out waiting for api-manager metrics service monitor: %v", err)
	}
}

// DaemonSetDefaultLogAnnotationTest checks that the deployed DS Pods have the
// default logging container set.
func DaemonSetDefaultLogAnnotationTest(t *testing.T, kubeclient kubernetes.Interface, ns string) {
	// DaemonSet will have already started, no need to wait.
	pods, err := kubeclient.CoreV1().Pods(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", k8s.AppComponent, deploy.DaemonSetName),
	})
	if err != nil {
		t.Fatalf("failed to get nodes: %v", err)
	}
	if len(pods.Items) == 0 {
		t.Fatal("expected StorageOS pods")
	}

	// Check the default log container annotation is set on all pods.
	for _, pod := range pods.Items {
		got, ok := pod.Annotations[deploy.DefaultLogsContainerAnnotationName]
		if !ok {
			t.Errorf("expected annotation %q not set on pod %q", deploy.DefaultLogsContainerAnnotationName, pod.Name)
		}
		if ok && got != deploy.NodeContainerName {
			t.Errorf("expected annotation %q set to %q on pod %q, want %q", deploy.DefaultLogsContainerAnnotationName, got, pod.Name, deploy.NodeContainerName)
		}
	}
}
