package nfsserver

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/storageos/cluster-operator/internal/pkg/image"
	storageosapis "github.com/storageos/cluster-operator/pkg/apis"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

//nolint // This function is shown as unused by the linter.
// getTestCluster returns a StorageOSCluster object with the given properties.
func getTestCluster(
	name string, namespace string,
	spec storageosv1.StorageOSClusterSpec,
	status storageosv1.StorageOSClusterStatus) *storageosv1.StorageOSCluster {
	return &storageosv1.StorageOSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   spec,
		Status: status,
	}
}

//nolint // This function is shown as unused by the linter.
// getTestNFSServer returns a NFSServer object with the given properties.
func getTestNFSServer(
	name string, namespace string,
	spec storageosv1.NFSServerSpec,
	status storageosv1.NFSServerStatus) *storageosv1.NFSServer {
	return &storageosv1.NFSServer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec:   spec,
		Status: status,
	}
}

func TestUpdateSpec(t *testing.T) {
	// This test used to work with the controller-runtime fake client. The fake
	// client has been deprecated and this test fails due to unexpected issues.
	// NFS Controller is no longer used in StorageOS v2. This test will be
	// removed with the NFS controller.
	t.Skip("skipping... fails with the controller-runtime fake client")

	emptyClusterSpec := storageosv1.StorageOSClusterSpec{}
	emptyClusterStatus := storageosv1.StorageOSClusterStatus{}
	emptyNFSSpec := storageosv1.NFSServerSpec{}
	emptyNFSStatus := storageosv1.NFSServerStatus{}

	testcases := []struct {
		name          string
		cluster       *storageosv1.StorageOSCluster
		nfsServer     *storageosv1.NFSServer
		wantNFSServer *storageosv1.NFSServer
		wantUpdate    bool
		wantErr       error
	}{
		{
			name:      "inherit attributes from cluster",
			cluster:   getTestCluster("cluster1", "default", emptyClusterSpec, emptyClusterStatus),
			nfsServer: getTestNFSServer("nfs1", "default", emptyNFSSpec, emptyNFSStatus),
			wantNFSServer: getTestNFSServer("nfs1", "default",
				storageosv1.NFSServerSpec{
					StorageClassName: "fast",
					NFSContainer:     image.DefaultNFSContainerImage,
				},
				emptyNFSStatus,
			),
			wantUpdate: true,
		},
		{
			// Check if the overridden cluster level defaults are inherited to
			// the NFS Server.
			name: "update the default properties in cluster",
			cluster: getTestCluster(
				"cluster1", "default",
				storageosv1.StorageOSClusterSpec{
					StorageClassName: "testsc",
					Images: storageosv1.ContainerImages{
						NFSContainer: "test-image",
					},
				}, emptyClusterStatus),
			nfsServer: getTestNFSServer("nfs1", "default", emptyNFSSpec, emptyNFSStatus),
			wantNFSServer: getTestNFSServer("nfs1", "default", storageosv1.NFSServerSpec{
				StorageClassName: "testsc",
				NFSContainer:     "test-image",
			}, emptyNFSStatus),
			wantUpdate: true,
		},
		{
			// Check that there's no update when the NFS Server CR is already
			// up-to-date.
			name:    "no new attributes to update",
			cluster: getTestCluster("cluster1", "default", emptyClusterSpec, emptyClusterStatus),
			nfsServer: getTestNFSServer(
				"nfs1", "default",
				storageosv1.NFSServerSpec{
					StorageClassName: "fast",
					NFSContainer:     image.DefaultNFSContainerImage,
				}, emptyNFSStatus),
			wantUpdate: false,
		},
		{
			// When the attributes are defined in NFS Server CR, no CR update
			// should happen.
			name:    "override default attributes",
			cluster: getTestCluster("cluster1", "default", emptyClusterSpec, emptyClusterStatus),
			nfsServer: getTestNFSServer("nfs1", "default", storageosv1.NFSServerSpec{
				StorageClassName: "testsc",
				NFSContainer:     "test-image",
			}, emptyNFSStatus),
			wantUpdate: false,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create a new scheme and add StorageOS APIs to it. Pass this to the
			// k8s client so that it can create StorageOS resources.
			testScheme := runtime.NewScheme()
			if err := storageosapis.AddToScheme(testScheme); err != nil {
				t.Fatal(err)
			}

			client := fake.NewFakeClientWithScheme(testScheme, tc.cluster, tc.nfsServer)

			reconciler := ReconcileNFSServer{
				client: client,
			}

			// Update NFSServer instance with the StorageOS Cluster and check the
			// results.
			result, err := reconciler.updateSpec(tc.nfsServer, tc.cluster)
			if err != nil {
				t.Fatalf("error while updating spec: %v", err)
			}

			if result != tc.wantUpdate {
				t.Errorf("unexpected update spec result:\n\t(WNT) %t\n\t(GOT) %t", tc.wantUpdate, result)
			}

			// If there was an update, get the NFS Server and check if it's as
			// expected.
			if tc.wantUpdate {
				namespacedNameNFS := types.NamespacedName{Name: tc.nfsServer.Name, Namespace: tc.nfsServer.Namespace}
				nfsServer := &storageosv1.NFSServer{}

				if err := client.Get(context.TODO(), namespacedNameNFS, nfsServer); err != nil {
					t.Fatalf("failed to get NFS Server: %v", err)
				}

				if !reflect.DeepEqual(nfsServer, tc.wantNFSServer) {
					t.Errorf("unexpected NFS Server:\n\t(WNT) %v\n\t(GOT) %v", tc.wantNFSServer, nfsServer)
				}
			}
		})
	}
}
