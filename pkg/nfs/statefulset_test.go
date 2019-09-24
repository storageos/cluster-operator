package nfs

import (
	"context"
	"testing"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestCreateStatefulSet(t *testing.T) {
	testcases := []struct {
		name                     string
		nfsServerSpec            storageosv1.NFSServerSpec
		wantVolumeClaimTemplates bool
		wantUseExistingVolume    bool
	}{
		{
			name:                     "default nfs server spec",
			nfsServerSpec:            storageosv1.NFSServerSpec{},
			wantVolumeClaimTemplates: true,
		},
		{
			name: "specify existing volume claim",
			nfsServerSpec: storageosv1.NFSServerSpec{
				PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "some-pvc",
				},
			},
			wantUseExistingVolume: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClient()

			// NFSServer config.
			nfsServer := &storageosv1.NFSServer{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-nfs-server",
					Namespace: "default",
				},
				Spec: tc.nfsServerSpec,
			}

			// StorageOS Cluster config.
			stosCluster := &storageosv1.StorageOSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "some-stos-cluster",
					Namespace: "default",
				},
				Spec: storageosv1.StorageOSClusterSpec{},
			}

			// NFSServer deployment.
			deployment := NewDeployment(client, stosCluster, nfsServer, nil, nil, nil)

			// Size of the NFS volume.
			size := resource.MustParse("1Gi")

			// Create statefulset, using random port numbers, not relevant here.
			err := deployment.createStatefulSet(&size, 5555, 6666)
			if err != nil {
				t.Errorf("unexpected error %v", err)
			}

			// Get the created statefulset.
			createdStatefulSet := &appsv1.StatefulSet{}
			nsName := types.NamespacedName{
				Name:      nfsServer.Name,
				Namespace: nfsServer.Namespace,
			}
			if err := client.Get(context.Background(), nsName, createdStatefulSet); err != nil {
				t.Fatal("failed to get the created statefulset", err)
			}

			// Get the total number of VolumeClaimTemplates and Pod Volumes.
			totalVCTemplates := len(createdStatefulSet.Spec.VolumeClaimTemplates)
			totalPodVolumes := len(createdStatefulSet.Spec.Template.Spec.Volumes)

			var wantTotalTemplates, wantPodVols int

			if tc.wantVolumeClaimTemplates {
				wantTotalTemplates = 1
			} else {
				wantTotalTemplates = 0
			}
			if totalVCTemplates != wantTotalTemplates {
				t.Errorf("unexpected number of VolumeClaimTemplates:\n\t(WNT) %d\n\t(GOT) %d", wantTotalTemplates, totalVCTemplates)
			}

			if tc.wantUseExistingVolume {
				wantPodVols = 2
			} else {
				wantPodVols = 1
			}
			if totalPodVolumes != wantPodVols {
				t.Errorf("unexpected number of pod volumes:\n\t(WNT) %d\n\t(GOT) %d", wantPodVols, totalPodVolumes)
			}
		})

	}
}
