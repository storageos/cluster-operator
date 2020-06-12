package storageoscluster

import (
	"context"
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	storageosapis "github.com/storageos/cluster-operator/pkg/apis"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

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

func TestGetCurrentStorageOSCluster(t *testing.T) {
	emptySpec := storageosv1.StorageOSClusterSpec{}
	emptyStatus := storageosv1.StorageOSClusterStatus{}

	// Create a new scheme and add the required schemes to it.
	scheme := runtime.NewScheme()
	if err := kscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := storageosapis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

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
			// Create fake client.
			client := fake.NewFakeClientWithScheme(scheme)

			// Create the clusters.
			for _, c := range tc.clusters {
				if err := client.Create(context.TODO(), c); err != nil {
					t.Fatalf("failed to create StorageOSCluster: %v", err)
				}
			}

			cc, err := GetCurrentStorageOSCluster(client)
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
