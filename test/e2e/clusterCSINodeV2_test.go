// +build v2

package e2e

import (
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	storageos "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	deploy "github.com/storageos/cluster-operator/pkg/storageos"
	testutil "github.com/storageos/cluster-operator/test/e2e/util"
)

// TestClusterCSIDeployment test the CSI helper deployment as Deployment.
func TestClusterCSINodeV2(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	resourceNS := "storageos"

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("could not get namespace: %v", err)
	}

	clusterSpec := storageos.StorageOSClusterSpec{
		SecretRefName:      "storageos-api",
		SecretRefNamespace: "default",
		Namespace:          resourceNS,
		CSI: storageos.StorageOSClusterCSI{
			Enable:             true,
			DeploymentStrategy: "deployment",
		},
		Tolerations: []corev1.Toleration{
			{
				Key:      "key",
				Operator: corev1.TolerationOpEqual,
				Value:    "value",
				Effect:   corev1.TaintEffectNoSchedule,
			},
		},
		K8sDistro: "openshift",
		Images: storageos.ContainerImages{
			NodeContainer: "rotsesgao/node:v2",
		},
		KVBackend: storageos.StorageOSClusterKVBackend{
			Address: "etcd-client.default.svc.cluster.local:2379",
		},
	}

	testStorageOS := testutil.NewStorageOSCluster(namespace, clusterSpec)

	testutil.SetupOperator(t, ctx)
	err = testutil.DeployCluster(t, ctx, testStorageOS)
	if err != nil {
		t.Fatal(err)
	}

	f := framework.Global

	// TODO - Status not confirmed working yet. Assume started after waiting
	// for the timeout for now.
	// err = f.Client.Get(goctx.TODO(), types.NamespacedName{Name: testutil.TestClusterCRName, Namespace: namespace}, testStorageOS)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	// testutil.ClusterStatusCheck(t, testStorageOS.Status, 1)

	daemonset, err := f.KubeClient.AppsV1().DaemonSets(resourceNS).Get("storageos-daemonset", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get storageos-daemonset: %v", err)
	}

	info, err := f.KubeClient.Discovery().ServerVersion()
	if err != nil {
		t.Fatalf("failed to get version info: %v", err)
	}

	version := strings.TrimLeft(info.String(), "v")

	//Check the number of containers in daemonset pod spec.
	if deploy.CSIV1Supported(version) {
		if len(daemonset.Spec.Template.Spec.Containers) != 3 {
			t.Errorf("unexpected number of daemonset pod containers:\n\t(GOT) %d\n\t(WNT) %d", len(daemonset.Spec.Template.Spec.Containers), 3)
		}
	} else {
		if len(daemonset.Spec.Template.Spec.Containers) != 2 {
			t.Errorf("unexpected number of daemonset pod containers:\n\t(GOT) %d\n\t(WNT) %d", len(daemonset.Spec.Template.Spec.Containers), 2)
		}
	}

	// Test StorageOSCluster CR attributes.
	testutil.StorageOSClusterCRAttributesTest(t, testutil.TestClusterCRName, namespace)

	// Test CSIDriver resource existence.
	testutil.CSIDriverResourceTest(t, deploy.StorageOSProvisionerName)

	// Test node label sync.
	// TODO: Currently relies on v1 CLI.
	// testutil.NodeLabelSyncTest(t, f.KubeClient)
}
