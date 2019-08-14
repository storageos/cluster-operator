package util

import (
	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// getStorageClass returns a generic StorageClass object with the given name.
func getStorageClass(name string) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
}

// DeleteStorageClass deletes a StorageClass resource, given a name.
func DeleteStorageClass(c client.Client, name string) error {
	return DeleteObject(c, getStorageClass(name))
}

// CreateStorageClass creates a StorageClass resource, given a name, provisioner
// and parameters.
func CreateStorageClass(c client.Client, name, provisioner string, params map[string]string) error {
	sc := getStorageClass(name)
	sc.Provisioner = provisioner
	sc.Parameters = params
	return CreateOrUpdateObject(c, sc)
}
