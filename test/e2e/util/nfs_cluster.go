package util

import (
	goctx "context"
	"fmt"
	"testing"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	storageos "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	nfs "github.com/storageos/cluster-operator/pkg/nfs"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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

	err = WaitForStatefulSet(t, f.KubeClient, nfsServer.Namespace, nfsServer.Name, RetryInterval, Timeout*2)
	if err != nil {
		t.Fatal(err)
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

	// Delete the NFS server.
	if err := f.Client.Delete(goctx.TODO(), testNFSServer); err != nil {
		t.Error("failed to delete NFS Server", err)
	}

	// Delete the PVC used by NFS server because it's not cleaned up when the
	// server is deleted.
	pvc := &corev1.PersistentVolumeClaim{}
	pvcNSName := types.NamespacedName{
		// PVC name format: <pvc-prefix>-<statefulset-pod-name>
		Name:      fmt.Sprintf("%s-%s-%s", nfs.PVCNamePrefix, nfsServerName, "0"),
		Namespace: defaultNS,
	}
	if err := f.Client.Get(goctx.TODO(), pvcNSName, pvc); err != nil {
		t.Error("failed to get PVC", err)
	}
	if err := f.Client.Delete(goctx.TODO(), pvc); err != nil {
		t.Error("failed to delete PVC", err)
	}
}
