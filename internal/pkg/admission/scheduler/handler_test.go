package scheduler

import (
	"context"
	"fmt"
	"testing"

	storageosapis "github.com/storageos/cluster-operator/pkg/apis"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestMutatePodFn(t *testing.T) {
	storageosSchedulerName := "storageos-scheduler"
	storageosCSIProvisioner := "storageos"
	storageosNativeProvisioner := "kubernetes.io/storageos"

	// Create a new scheme and add all the types from different clientsets.
	scheme := runtime.NewScheme()
	kscheme.AddToScheme(scheme)
	apiextensionsv1beta1.AddToScheme(scheme)
	storageosapis.AddToScheme(scheme)

	// StorageOS StorageClass.
	stosSC := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fast",
		},
		Provisioner: storageosCSIProvisioner,
	}

	// StorageOS StorageClass with different provisioner.
	stosNativeSC := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fast2",
		},
		Provisioner: storageosNativeProvisioner,
	}

	// Non-StorageOS StorageClass.
	fooSC := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "slow",
		},
		Provisioner: "foo-provisioner",
	}

	// PVC that uses StorageOS StorageClass.
	stosPVC := createPVC("pv1", "default", stosSC.Name)

	// PVC that uses StorageOS native StorageClass.
	stosNativePVC := createPVC("pv2", "default", stosNativeSC.Name)

	// PVC that uses non-StorageOS StorageClass.
	fooPVC := createPVC("pv3", "default", fooSC.Name)

	testcases := []struct {
		name              string
		volumeClaimNames  []string
		schedulerDisabled bool
		wantSchedulerName string
	}{
		{
			name:              "pod with storageos volume, scheduler enabled",
			volumeClaimNames:  []string{stosPVC.Name},
			schedulerDisabled: false,
			wantSchedulerName: storageosSchedulerName,
		},
		{
			name:              "pod with storageos volume, scheduler disabled",
			volumeClaimNames:  []string{stosPVC.Name},
			schedulerDisabled: true,
		},
		{
			name:              "pod without storageos volume, scheduler enabled",
			volumeClaimNames:  []string{fooPVC.Name},
			schedulerDisabled: false,
		},
		{
			name:              "pod without storageos volume, scheduler disabled",
			volumeClaimNames:  []string{fooPVC.Name},
			schedulerDisabled: true,
		},
		{
			// Using the PVC that uses the native provisioner StorageClass.
			name:              "pod with non-storageos and storageos volumes, scheduler enabled",
			volumeClaimNames:  []string{stosNativePVC.Name, fooPVC.Name},
			schedulerDisabled: false,
			wantSchedulerName: storageosSchedulerName,
		},
		{
			name:              "pod with non-storageos and storageos volumes, scheduler disabled",
			volumeClaimNames:  []string{stosPVC.Name, fooPVC.Name},
			schedulerDisabled: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// StorageOS Cluster with scheduler configured.
			stosCluster := &storageosv1.StorageOSCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "stos",
					Namespace: "default",
				},
				Spec: storageosv1.StorageOSClusterSpec{
					DisableScheduler: tc.schedulerDisabled,
				},
			}

			// Pod that uses PVCs.
			pod := &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod1",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{},
					Containers: []corev1.Container{
						{
							Name:  "some-app",
							Image: "some-image",
						},
					},
				},
			}

			// Append the volumes in the pod spec.
			for i, claimName := range tc.volumeClaimNames {
				pod.Spec.Volumes = append(pod.Spec.Volumes, corev1.Volume{
					Name: fmt.Sprintf("vol%d", i),
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: claimName,
						},
					},
				})
			}

			// Create all the above resources and get a k8s client.
			client := fake.NewFakeClientWithScheme(scheme, stosCluster, stosSC, stosNativeSC, fooSC, stosPVC, stosNativePVC, fooPVC, pod)

			// Create a PodSchedulerSetter instance with the fake client.
			podSchedulerSetter := PodSchedulerSetter{
				client: client,
				Provisioners: []string{
					storageosCSIProvisioner,
					storageosNativeProvisioner,
				},
				SchedulerName: storageosSchedulerName,
			}

			// Pass the created pod to the mutatePodFn and check if the schedulerName in
			// podSpec changed.
			if err := podSchedulerSetter.mutatePodsFn(context.Background(), pod); err != nil {
				t.Fatalf("failed to mutate pod: %v", err)
			}

			if pod.Spec.SchedulerName != tc.wantSchedulerName {
				t.Errorf("unexpected pod scheduler name:\n\t(WNT) %s\n\t(GOT) %s", tc.wantSchedulerName, pod.Spec.SchedulerName)
			}
		})
	}
}

// createPVC creates and returns a PVC object.
func createPVC(name, namespace, storageClassName string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			StorageClassName: &storageClassName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
}
