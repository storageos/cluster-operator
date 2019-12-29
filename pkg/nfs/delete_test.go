package nfs

import (
	"context"
	"testing"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestDelete(t *testing.T) {
	// Existing PVC that can be used with a NFS Server.
	existingPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-pvc",
			Namespace: "default",
		},
		Spec: corev1.PersistentVolumeClaimSpec{},
	}

	testcases := []struct {
		name                  string
		nfsServerSpec         storageosv1.NFSServerSpec
		wantDefaultPVCDeleted bool
	}{
		{
			name: "delete dynamically created volume",
			nfsServerSpec: storageosv1.NFSServerSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
			wantDefaultPVCDeleted: true,
		},
		{
			// NFS Server spec reclaim policy must not be respected.
			name: "volume reclaim policy - retain",
			nfsServerSpec: storageosv1.NFSServerSpec{
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
				PersistentVolumeReclaimPolicy: corev1.PersistentVolumeReclaimDelete,
			},
			wantDefaultPVCDeleted: true,
		},
		{
			// Existing volume must not be deleted.
			name: "specify existing volume claim",
			nfsServerSpec: storageosv1.NFSServerSpec{
				PersistentVolumeClaim: corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: existingPVC.Name,
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClient(existingPVC)
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

			// Default dynamically created PVC for NFS Server.
			createdPVC := &corev1.PersistentVolumeClaim{}
			pvcNSName := types.NamespacedName{
				Name:      nfsServer.Name,
				Namespace: nfsServer.Namespace,
			}

			// If PVC volume source is not specified, check if the default PVC
			// is created.
			if nfsServer.Spec.PersistentVolumeClaim.ClaimName == "" {
				if err := client.Get(context.Background(), pvcNSName, createdPVC); err != nil {
					t.Fatalf("failed to get the created PVC: %v", err)
				}
			}

			// Delete NFS Server.
			if err := deployment.Delete(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			// If PVC Volume Source was not provided, check their existence.
			if nfsServer.Spec.PersistentVolumeClaim.ClaimName == "" {
				// Get the default PVC.
				err := client.Get(context.Background(), pvcNSName, createdPVC)

				// If the PVC is expected to be deleted, the error must be NOT
				// FOUND.
				if tc.wantDefaultPVCDeleted {
					if !errors.IsNotFound(err) {
						t.Errorf("expected the PVC to be deleted")
					}
				} else {
					if err != nil {
						t.Errorf("expected the default volume to exist")
					}
				}
			} else {
				// Ensure that the existing provided PVC has not been deleted.
				existingPVCNSName := types.NamespacedName{
					Name:      existingPVC.Name,
					Namespace: existingPVC.Namespace,
				}
				if err := client.Get(context.Background(), existingPVCNSName, existingPVC); err != nil {
					t.Error("expected existing PVC to not be deleted")
				}
			}
		})
	}
}
