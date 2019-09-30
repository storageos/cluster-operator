package nfsserver

import (
	"reflect"
	"testing"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	fakeStosClientset "github.com/storageos/cluster-operator/pkg/client/clientset/versioned/fake"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

func TestGetCurrentStorageOSCluster(t *testing.T) {
	emptySpec := storageosv1.StorageOSClusterSpec{}
	emptyStatus := storageosv1.StorageOSClusterStatus{}

	testcases := []struct {
		name            string
		clusters        []*storageosv1.StorageOSCluster
		wantClusterName string
		wantErr         error
	}{
		{
			name: "multiple clusters with one ready",
			clusters: []*storageosv1.StorageOSCluster{
				getTestCluster("cluster1", "default", emptySpec, emptyStatus),
				getTestCluster("cluster2", "foo", emptySpec,
					storageosv1.StorageOSClusterStatus{
						Phase: storageosv1.ClusterPhaseRunning,
					}),
				getTestCluster("cluster3", "default", emptySpec, emptyStatus),
			},
			wantClusterName: "cluster2",
		},
		{
			name: "multiple clusters with none ready",
			clusters: []*storageosv1.StorageOSCluster{
				getTestCluster("cluster1", "default", emptySpec, emptyStatus),
				getTestCluster("cluster2", "default", emptySpec,
					storageosv1.StorageOSClusterStatus{
						Phase: storageosv1.ClusterPhaseInitial,
					}),
				getTestCluster("cluster3", "default", emptySpec, emptyStatus),
			},
			wantErr: ErrNoCluster,
		},
		{
			name: "single cluster not ready",
			clusters: []*storageosv1.StorageOSCluster{
				getTestCluster("cluster1", "default", emptySpec, emptyStatus),
			},
			wantClusterName: "cluster1",
		},
		{
			name:    "no cluster",
			wantErr: ErrNoCluster,
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			// Create fake storageos client.
			stosClient := fakeStosClientset.NewSimpleClientset()

			// Create the clusters.
			for _, c := range tc.clusters {
				_, err := stosClient.StorageosV1().StorageOSClusters(c.Namespace).Create(c)
				if err != nil {
					t.Fatalf("failed to create StorageOSCluster: %v", err)
				}
			}

			// Create a reconciler.
			reconciler := ReconcileNFSServer{
				stosClientset: stosClient,
			}

			cc, err := reconciler.getCurrentStorageOSCluster()
			if err != nil {
				if !reflect.DeepEqual(tc.wantErr, err) {
					t.Fatalf("unexpected error while getting current cluster: %v", err)
				}
			} else {
				if tc.wantClusterName != cc.Name {
					t.Errorf("unexpected current cluster selection:\n\t(WNT) %s\n\t(GOT) %s", tc.wantClusterName, cc.Name)
				}
			}
		})
	}
}
