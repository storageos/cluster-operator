package util

import (
	"context"
	goctx "context"
	"testing"
	"time"

	"github.com/blang/semver"
	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageos "github.com/storageos/cluster-operator/pkg/storageos"
)

const (
	// defaultStorageClassKey is the annotation used to denote whether a
	// StorageClass is the cluster default.
	defaultStorageClassKey = "storageclass.kubernetes.io/is-default-class"

	deleteRetryInterval = time.Second
	deleteRetryTimeout  = 20 * time.Second
)

// PodSchedulerAdmissionControllerTest checks if the pod scheduler mutating
// admission controller mutates the scheduler name of a pod by creates a pvc
// backed by StorageOS and a pod that uses the PVC.
// NOTE: This test has a minimum k8s version requirement.
func PodSchedulerAdmissionControllerTest(t *testing.T, ctx *framework.Context) {
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

	scName1 := "sc1"
	scName2 := "sc2"

	tests := []struct {
		testname          string
		storageClasses    []*storagev1.StorageClass
		pvcs              []*corev1.PersistentVolumeClaim
		pod               *corev1.Pod
		wantSchedulerName string
	}{
		{
			testname: "one storageos pvc",
			storageClasses: []*storagev1.StorageClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName1,
					},
					Provisioner: storageos.StorageOSProvisionerName,
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &scName1,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
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
				},
			},
			wantSchedulerName: storageos.SchedulerExtenderName,
		},
		{
			testname: "one non-storageos pvc",
			storageClasses: []*storagev1.StorageClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName1,
					},
					Provisioner: "foo",
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &scName1,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
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
				},
			},
			wantSchedulerName: "default-scheduler",
		},
		{
			testname: "mixed pvcs",
			storageClasses: []*storagev1.StorageClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName1,
					},
					Provisioner: storageos.StorageOSProvisionerName,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName2,
					},
					Provisioner: "foo",
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc1",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &scName1,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc2",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &scName2,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "some-data1",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "some-pvc1",
								},
							},
						},
						{
							Name: "some-data2",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "some-pvc2",
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
									Name:      "some-data1",
									MountPath: "/data1",
								},
								{
									Name:      "some-data2",
									MountPath: "/data2",
								},
							},
						},
					},
				},
			},
			wantSchedulerName: "storageos-scheduler",
		},
		{
			testname: "mixed pvcs with default non-storageos ",
			storageClasses: []*storagev1.StorageClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName1,
					},
					Provisioner: storageos.StorageOSProvisionerName,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName2,
						Annotations: map[string]string{
							"storageclass.kubernetes.io/is-default-class": "true",
						},
					},
					Provisioner: "foo",
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc1",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &scName1,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc2",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "some-data1",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "some-pvc1",
								},
							},
						},
						{
							Name: "some-data2",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "some-pvc2",
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
									Name:      "some-data1",
									MountPath: "/data1",
								},
								{
									Name:      "some-data2",
									MountPath: "/data2",
								},
							},
						},
					},
				},
			},
			wantSchedulerName: "storageos-scheduler",
		},
		{
			testname: "mixed pvcs with default storageos ",
			storageClasses: []*storagev1.StorageClass{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName1,
						Annotations: map[string]string{
							"storageclass.kubernetes.io/is-default-class": "true",
						},
					},
					Provisioner: storageos.StorageOSProvisionerName,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: scName2,
					},
					Provisioner: "foo",
				},
			},
			pvcs: []*corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc1",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "some-pvc2",
						Namespace: "default",
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes:      []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
						StorageClassName: &scName2,
						Resources: corev1.ResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: resource.MustParse("1Gi"),
							},
						},
					},
				},
			},
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: "some-data1",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "some-pvc1",
								},
							},
						},
						{
							Name: "some-data2",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: "some-pvc2",
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
									Name:      "some-data1",
									MountPath: "/data1",
								},
								{
									Name:      "some-data2",
									MountPath: "/data2",
								},
							},
						},
					},
				},
			},
			wantSchedulerName: "storageos-scheduler",
		},
	}

	for _, tt := range tests {
		var tt = tt
		t.Run(tt.testname, func(t *testing.T) {
			f := framework.Global

			// If there is an existing default StorageClass, remove the annotation and
			// replace it after the test.
			revert, err := DisableDefaultStorageClass(t, f.Client)
			if err != nil {
				t.Fatalf("failed to disable default StorageClass: %v", err)
			}
			defer func() {
				if err := revert(); err != nil {
					t.Errorf("failed to revert disabled default StorageClass: %v", err)
				}
			}()

			// Create storage classes.
			for _, sc := range tt.storageClasses {
				var sc = sc
				if err := f.Client.Create(goctx.TODO(), sc, nil); err != nil {
					t.Fatalf("failed to create sc: %v", err)
				}
				t.Logf("created sc: %s", sc.Name)
				defer func() {
					t.Logf("deleting sc: %s", sc.Name)
					if err := WaitForDelete(t, f.Client, types.NamespacedName{Name: sc.Name}, sc, deleteRetryInterval, deleteRetryTimeout); err != nil {
						t.Errorf("failed to delete sc: %v", err)
					}
				}()
			}

			// Create pvcs.
			for _, pvc := range tt.pvcs {
				var pvc = pvc
				if err := f.Client.Create(goctx.TODO(), pvc, nil); err != nil {
					t.Fatalf("failed to create pvc using StorageOS: %v", err)
				}
				t.Logf("created pvc: %s/%s", pvc.Namespace, pvc.Name)
				defer func() {
					t.Logf("deleting pvc: %s/%s", pvc.Namespace, pvc.Name)
					if err := WaitForDelete(t, f.Client, types.NamespacedName{Name: pvc.Name, Namespace: pvc.Namespace}, pvc, deleteRetryInterval, deleteRetryTimeout); err != nil {
						t.Errorf("failed to delete pvc: %v", err)
					}
				}()
			}

			// Wait for the pvcs to be created.
			time.Sleep(2 * time.Second)

			if err := f.Client.Create(goctx.TODO(), tt.pod, nil); err != nil {
				t.Errorf("failed to create pod using StorageOS: %v", err)
			}
			t.Logf("created pod: %s/%s", tt.pod.Namespace, tt.pod.Name)
			defer func() {
				t.Logf("deleting pod: %s/%s", tt.pod.Namespace, tt.pod.Name)
				if err := WaitForDelete(t, f.Client, types.NamespacedName{Name: tt.pod.Name, Namespace: tt.pod.Namespace}, tt.pod, deleteRetryInterval, deleteRetryTimeout); err != nil {
					t.Errorf("failed to delete pod: %v", err)
				}
			}()

			// Wait for the pod to be created.
			time.Sleep(5 * time.Second)

			// Get the pod and check the pod scheduler name.
			var pod corev1.Pod
			if err := f.Client.Get(goctx.TODO(), types.NamespacedName{Name: tt.pod.Name, Namespace: tt.pod.Namespace}, &pod); err != nil {
				t.Errorf("failed to get pod using storageos: %v", err)
			}
			if pod.Spec.SchedulerName != tt.wantSchedulerName {
				t.Errorf("unexpected scheduler name:\n\t(WNT) %s\n\t(GOT) %s", tt.wantSchedulerName, pod.Spec.SchedulerName)
			}

			if t.Failed() {
				logs, err := GetAPIManagerLogs()
				if err != nil {
					t.Error(errors.Wrap(err, "failed to fetch logs"))
				}
				t.Log(logs)
			}
		})
	}
}

// WaitForDelete deletes an object and waits for it to be removed.
func WaitForDelete(t *testing.T, fc framework.FrameworkClient, key types.NamespacedName, obj runtime.Object, retryInterval, timeout time.Duration) error {
	if err := fc.Delete(goctx.TODO(), obj); err != nil {
		t.Errorf("failed to delete %s %s/%s: %v", obj.GetObjectKind().GroupVersionKind().Group, key.Namespace, key.Name, err)
	}

	err := wait.Poll(retryInterval, timeout, func() (done bool, err error) {
		err = fc.Get(context.TODO(), key, obj)
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}

		t.Logf("waiting for removal of %s %s/%s", obj.GetObjectKind().GroupVersionKind().Group, key.Namespace, key.Name)
		return false, nil
	})
	if err != nil {
		return err
	}
	t.Logf("deleted %s %s/%s", obj.GetObjectKind().GroupVersionKind().Group, key.Namespace, key.Name)
	return nil
}

// DisableDefaultStorageClass disables the default StorageClass, if set.  It
// returns a function that re-enables it.
func DisableDefaultStorageClass(t *testing.T, fc framework.FrameworkClient) (func() error, error) {
	scList := &storagev1.StorageClassList{}
	if err := fc.List(goctx.TODO(), scList, &client.ListOptions{}); err != nil {
		return nil, err
	}
	for _, sc := range scList.Items {
		var sc = sc
		if val, ok := sc.Annotations[defaultStorageClassKey]; ok && val == "true" {
			var orig = sc
			delete(sc.Annotations, defaultStorageClassKey)
			if err := fc.Update(goctx.TODO(), &sc); err != nil {
				return nil, err
			}
			revert := func() error {
				var sc storagev1.StorageClass
				err := fc.Get(goctx.TODO(), types.NamespacedName{Name: orig.Name}, &sc)
				if err != nil {
					return err
				}
				if sc.Annotations == nil {
					sc.Annotations = make(map[string]string)
				}
				sc.Annotations[defaultStorageClassKey] = "true"
				return fc.Update(goctx.TODO(), &sc)
			}
			return revert, nil
		}
	}
	// No default StorageClass.
	return func() error { return nil }, nil
}
