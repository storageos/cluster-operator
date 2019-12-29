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
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDeploy(t *testing.T) {
	testcases := []struct {
		name          string
		nfsServerSpec storageosv1.NFSServerSpec
	}{
		{
			name: "default nfs server spec",
			nfsServerSpec: storageosv1.NFSServerSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		},
		{
			name: "specify existing volume claim",
			nfsServerSpec: storageosv1.NFSServerSpec{
				PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "some-pvc",
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClient()
			kConfig := &rest.Config{}

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
			deployment := NewDeployment(client, kConfig, stosCluster, nfsServer, nil, nil, nil)

			// Deploy NFS Server.
			if err := deployment.Deploy(); err != nil {
				t.Errorf("unexpected error while deploying: %v", err)
			}

			// If PVC volume source is not specified, check if the default PVC
			// is created.
			if nfsServer.Spec.PersistentVolumeClaim.ClaimName == "" {
				createdPVC := &corev1.PersistentVolumeClaim{}
				pvcNSName := types.NamespacedName{
					Name:      nfsServer.Name,
					Namespace: nfsServer.Namespace,
				}
				if err := client.Get(context.Background(), pvcNSName, createdPVC); err != nil {
					t.Fatalf("failed to get the created PVC: %v", err)
				}
			}

			// Check if the StatefulSet was created.
			createdStatefulSet := &appsv1.StatefulSet{}
			ssNSName := types.NamespacedName{
				Name:      nfsServer.Name,
				Namespace: nfsServer.Namespace,
			}
			if err := client.Get(context.Background(), ssNSName, createdStatefulSet); err != nil {
				t.Fatalf("failed to get the created statefulset: %v", err)
			}
		})
	}
}
