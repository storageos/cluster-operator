package util

import (
	goctx "context"
	"testing"
	"time"

	"github.com/blang/semver"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	storageos "github.com/storageos/cluster-operator/pkg/storageos"
)

// PodSchedulerAdmissionControllerTest checks if the pod scheduler mutating
// admission controller mutates the scheduler name of a pod by creates a pvc
// backed by StorageOS and a pod that uses the PVC.
// NOTE: This test has a minimum k8s version requirement.
func PodSchedulerAdmissionControllerTest(t *testing.T, ctx *framework.TestCtx) {
	k8sVerMajor := 1
	k8sVerMinor := 13
	k8sVerPatch := 0
	// Minimum version of k8s required to run this test.
	minVersion := semver.Version{
		Major: uint64(k8sVerMajor),
		Minor: uint64(k8sVerMinor),
		Patch: uint64(k8sVerPatch),
	}

	// Check the k8s version before running this test. Admission controller
	// does not works on openshift 3.11 (k8s 1.11).
	featureSupported, err := featureSupportAvailable(minVersion)
	if err != nil {
		t.Errorf("failed to check platform support for admission controller test: %v", err)
		return
	}

	// Skip if the feature is not supported.
	if !featureSupported {
		return
	}

	f := framework.Global

	// Provide some time for StorageOS initialization to be complete.
	time.Sleep(10 * time.Second)

	// Create a StorageOS PVC.
	scName := storageosv1.DefaultStorageClassName
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "some-pvc",
			Namespace: "default",
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			StorageClassName: &scName,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse("1Gi"),
				},
			},
		},
	}
	if err := f.Client.Create(goctx.TODO(), pvc, &framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval}); err != nil {
		t.Fatalf("failed to create pvc using StorageOS: %v", err)
	}

	// Wait for the volume to be created.
	time.Sleep(5 * time.Second)

	// Create a Pod with the above PVC.
	podSpec := corev1.PodSpec{
		Volumes: []corev1.Volume{
			{
				Name: "some-data",
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: "some-pvc",
					},
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:  "test-app",
				Image: "nginx",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "some-data",
						MountPath: "/data",
					},
				},
			},
		},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-app",
			Namespace: "default",
		},
		Spec: podSpec,
	}
	if err := f.Client.Create(goctx.TODO(), pod, &framework.CleanupOptions{TestContext: ctx, Timeout: CleanupTimeout, RetryInterval: CleanupRetryInterval}); err != nil {
		t.Fatalf("failed to create pod using StorageOS: %v", err)
	}

	// Wait for the pod to be created.
	time.Sleep(15 * time.Second)

	// Get the pod and check the pod scheduler name.
	if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: "test-app", Namespace: "default"}, pod); err != nil {
		t.Errorf("failed to get pod using storageos: %v", err)
	}
	if pod.Spec.SchedulerName != storageos.SchedulerExtenderName {
		t.Errorf("unexpected scheduler name:\n\t(WNT) %s\n\t(GOT) %s", storageos.SchedulerExtenderName, pod.Spec.SchedulerName)
	}
}
