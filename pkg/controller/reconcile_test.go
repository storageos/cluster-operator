package controller

import (
	"testing"

	clusterv1alpha1 "github.com/storageos/cluster-operator/pkg/apis/cluster/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSetCurrentClusterIfNone(t *testing.T) {
	cc := &ClusterController{}

	cluster1 := &clusterv1alpha1.StorageOSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-ns",
		},
	}
	cc.SetCurrentClusterIfNone(cluster1)

	if !cc.IsCurrentCluster(cluster1) {
		t.Error("failed to set current cluster")
	}

	cluster2 := cluster1.DeepCopy()
	cluster2.ObjectMeta.SetName("test-cluster2")

	cc.SetCurrentClusterIfNone(cluster2)
	if cc.IsCurrentCluster(cluster2) {
		t.Error("should not set current cluster if already set")
	}
}

func TestIsCurrentCluster(t *testing.T) {
	cluster1 := &clusterv1alpha1.StorageOSCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-cluster",
			Namespace: "test-ns",
		},
	}

	testcases := []struct {
		name       string
		cluster1   *clusterv1alpha1.StorageOSCluster
		cluster2   *clusterv1alpha1.StorageOSCluster
		wantResult bool
	}{
		{
			name:       "same cluster object comparison",
			cluster1:   cluster1,
			cluster2:   cluster1.DeepCopy(),
			wantResult: true,
		},
		{
			name:       "nil cluster object comparison",
			cluster1:   cluster1,
			cluster2:   nil,
			wantResult: false,
		},
		{
			name:     "same cluster object comparison",
			cluster1: cluster1,
			cluster2: &clusterv1alpha1.StorageOSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster2",
					Namespace: "test-ns",
				},
			},
			wantResult: false,
		},
		{
			name:     "same cluster object, different namespace comparison",
			cluster1: cluster1,
			cluster2: &clusterv1alpha1.StorageOSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-cluster",
					Namespace: "test-ns2",
				},
			},
			wantResult: false,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			cc := &ClusterController{}

			cc.SetCurrentClusterIfNone(tc.cluster1)

			result := cc.IsCurrentCluster(tc.cluster2)
			if result != tc.wantResult {
				t.Errorf("unexpected IsCurrentCluster result:\n\t(GOT) %v\n\t(WNT) %v", result, tc.wantResult)
			}
		})
	}
}
