package scheduler

import (
	"testing"

	storageosapis "github.com/storageos/cluster-operator/pkg/apis"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPodSchedulerSetter_IsManagedVolume(t *testing.T) {
	// Test values only.
	storageosSchedulerName := "storageos-scheduler"
	storageosCSIProvisioner := "storageos"
	storageosNativeProvisioner := "kubernetes.io/storageos"
	schedulerAnnotationKey := "storageos.com/scheduler"

	// Create a new scheme and add all the types from different clientsets.
	scheme := runtime.NewScheme()
	if err := kscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := apiextensionsv1beta1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := storageosapis.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

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

	// StorageOS StorageClass set as default.
	stosSCdefault := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fast",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		Provisioner: storageosCSIProvisioner,
	}

	// Non-StorageOS StorageClass.
	fooSCdefault := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "slow",
			Annotations: map[string]string{
				"storageclass.kubernetes.io/is-default-class": "true",
			},
		},
		Provisioner: "foo-provisioner",
	}

	testNamespace := "default"

	tests := []struct {
		name           string
		storageClasses []*storagev1.StorageClass
		pvc            *corev1.PersistentVolumeClaim
		volume         *corev1.Volume
		want           bool
		wantErr        bool
	}{
		{
			name:           "storageos volume",
			storageClasses: []*storagev1.StorageClass{stosSC, stosNativeSC, fooSC},
			pvc:            createPVC("pv1", testNamespace, stosSC.Name, false),
			volume:         createVolume("pv1"),
			want:           true,
		},
		{
			name:           "storageos volume, beta annotation",
			storageClasses: []*storagev1.StorageClass{stosSC, stosNativeSC, fooSC},
			pvc:            createPVC("pv1", testNamespace, stosSC.Name, true),
			volume:         createVolume("pv1"),
			want:           true,
		},
		{
			name:           "storageos native driver volume",
			storageClasses: []*storagev1.StorageClass{stosSC, stosNativeSC, fooSC},
			pvc:            createPVC("pv1", testNamespace, stosNativeSC.Name, false),
			volume:         createVolume("pv1"),
			want:           true,
		},
		{
			name:           "storageos native driver volume, beta annotation",
			storageClasses: []*storagev1.StorageClass{stosSC, stosNativeSC, fooSC},
			pvc:            createPVC("pv1", testNamespace, stosNativeSC.Name, true),
			volume:         createVolume("pv1"),
			want:           true,
		},
		{
			name:           "storageos volume, default storage class",
			storageClasses: []*storagev1.StorageClass{stosSCdefault, fooSC},
			pvc:            createPVC("pv1", testNamespace, "", false),
			volume:         createVolume("pv1"),
			want:           true,
		},
		{
			name:           "non-storageos volume",
			storageClasses: []*storagev1.StorageClass{stosSC, stosNativeSC, fooSC},
			pvc:            createPVC("pv1", testNamespace, fooSC.Name, false),
			volume:         createVolume("pv1"),
			want:           false,
		},
		{
			name:           "non-storageos volume, beta annotation",
			storageClasses: []*storagev1.StorageClass{stosSC, stosNativeSC, fooSC},
			pvc:            createPVC("pv1", testNamespace, fooSC.Name, true),
			volume:         createVolume("pv1"),
			want:           false,
		},
		{
			name:           "non-storageos volume, default storage class",
			storageClasses: []*storagev1.StorageClass{stosSC, stosNativeSC, fooSCdefault},
			pvc:            createPVC("pv1", testNamespace, "", true),
			volume:         createVolume("pv1"),
			want:           false,
		},
		{
			name:           "non-storageos volume, no storage classes",
			storageClasses: []*storagev1.StorageClass{},
			pvc:            createPVC("pv1", testNamespace, "", true),
			volume:         createVolume("pv1"),
			want:           false,
			wantErr:        true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Create all the above resources and get a k8s client.
			var objects []runtime.Object
			for _, sc := range tt.storageClasses {
				objects = append(objects, sc)
			}
			objects = append(objects, tt.pvc)
			client := fake.NewFakeClientWithScheme(scheme, objects...)

			// Create a PodSchedulerSetter instance with the fake client.
			podSchedulerSetter := PodSchedulerSetter{
				client: client,
				Provisioners: []string{
					storageosCSIProvisioner,
					storageosNativeProvisioner,
				},
				SchedulerName:          storageosSchedulerName,
				SchedulerAnnotationKey: schedulerAnnotationKey,
			}

			got, err := podSchedulerSetter.IsManagedVolume(*tt.volume, testNamespace)
			if (err != nil) != tt.wantErr {
				t.Errorf("PodSchedulerSetter.IsManagedVolume() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("PodSchedulerSetter.IsManagedVolume() = %v, want %v", got, tt.want)
			}
		})
	}
}

// createVolume creates and returns a Volume object.
func createVolume(pvcName string) *corev1.Volume {
	return &corev1.Volume{
		Name: pvcName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: pvcName,
			},
		},
	}
}
