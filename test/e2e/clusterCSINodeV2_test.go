// +build v2

package e2e

import (
	"context"
	"strings"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	storageos "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	deploy "github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/test/e2e/util"
	testutil "github.com/storageos/cluster-operator/test/e2e/util"
)

// TestClusterCSIDeployment test the CSI helper deployment as Deployment.
func TestClusterCSINodeV2(t *testing.T) {
	ctx := framework.NewTestCtx(t)
	defer ctx.Cleanup()
	resourceNS := "kube-system"

	namespace, err := ctx.GetNamespace()
	if err != nil {
		t.Fatalf("could not get namespace: %v", err)
	}

	clusterSpec := storageos.StorageOSClusterSpec{
		SecretRefName:      "storageos-api",
		SecretRefNamespace: "default",
		Namespace:          resourceNS,
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
			NodeContainer:       "rotsesgao/node:v2",
			APIManagerContainer: "storageos/api-manager:develop",
		},
		KVBackend: storageos.StorageOSClusterKVBackend{
			Address: "etcd-client.default.svc.cluster.local:2379",
		},
	}

	clusterNsName := types.NamespacedName{
		Name:      testutil.TestClusterCRName,
		Namespace: namespace,
	}
	testStorageOS := testutil.NewStorageOSCluster(clusterNsName, clusterSpec)

	t.Run("SetupOperator", func(t *testing.T) {
		testutil.SetupOperator(t, ctx)
	})
	t.Run("DeployCluster", func(t *testing.T) {
		err = testutil.DeployCluster(t, ctx, testStorageOS)
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ClusterStatusCheck", func(t *testing.T) {
		if err = testutil.ClusterStatusCheck(t, clusterNsName, 1, testutil.RetryInterval, testutil.Timeout); err != nil {
			logs, logErr := util.GetOperatorLogs(namespace)
			if logErr != nil {
				t.Error(errors.Wrap(logErr, "failed to fetch operator logs"))
			}
			t.Log(logs)
			t.Fatal(err)
		}
	})

	f := framework.Global

	daemonset, err := f.KubeClient.AppsV1().DaemonSets(resourceNS).Get(context.TODO(), "storageos-daemonset", metav1.GetOptions{})
	if err != nil {
		t.Fatalf("failed to get storageos-daemonset: %v", err)
	}

	info, err := f.KubeClient.Discovery().ServerVersion()
	if err != nil {
		t.Fatalf("failed to get version info: %v", err)
	}

	version := strings.TrimLeft(info.String(), "v")

	t.Run("CSIHelperCountTest", func(t *testing.T) {
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
	})

	// Test DaemonSet configuration.
	t.Run("DaemonSetDefaultLogAnnotationTest", func(t *testing.T) {
		testutil.DaemonSetDefaultLogAnnotationTest(t, f.KubeClient, resourceNS)
	})

	// Test StorageOSCluster CR attributes.
	t.Run("StorageOSClusterCRAttributesTest", func(t *testing.T) {
		testutil.StorageOSClusterCRAttributesTest(t, testutil.TestClusterCRName, namespace)
	})

	// Test CSIDriver resource existence.
	t.Run("CSIDriverResourceTest", func(t *testing.T) {
		testutil.CSIDriverResourceTest(t, deploy.StorageOSProvisionerName)
	})

	// API Manager tests.
	t.Run("APIManagerDeploymentTest", func(t *testing.T) {
		testutil.APIManagerDeploymentTest(t, resourceNS, testutil.RetryInterval, testutil.Timeout)
	})
	t.Run("APIManagerMetricsServiceTest", func(t *testing.T) {
		testutil.APIManagerMetricsServiceTest(t, resourceNS, testutil.RetryInterval, testutil.Timeout)
	})
	t.Run("APIManagerMetricsServiceMonitorTest", func(t *testing.T) {
		testutil.APIManagerMetricsServiceMonitorTest(t, resourceNS, testutil.RetryInterval, testutil.Timeout)
	})
	t.Run("APIManagerWebhookServiceTest", func(t *testing.T) {
		testutil.APIManagerMetricsServiceTest(t, resourceNS, testutil.RetryInterval, testutil.Timeout)
	})

	// Test pod scheduler mutating admission contoller.
	t.Run("PodSchedulerAdmissionControllerTest", func(t *testing.T) {
		testutil.PodSchedulerAdmissionControllerTest(t, ctx)
	})

	// Test StorageOSCluster CR mutation.
	t.Run("StorageOSClusterUpdateTest", func(t *testing.T) {
		testutil.StorageOSClusterUpdateTest(t, ctx)
	})

}
