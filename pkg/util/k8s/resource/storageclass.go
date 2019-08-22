package resource

import (
	"context"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StorageClassKind is the name of k8s StorageClass resource kind.
const StorageClassKind = "StorageClass"

// StorageClass implements k8s.Resource interface for k8s StorageClass resource.
type StorageClass struct {
	types.NamespacedName
	labels      map[string]string
	client      client.Client
	provisioner string
	params      map[string]string
}

// NewStorageClass returns an initialized StorageClass.
func NewStorageClass(
	c client.Client,
	name string,
	labels map[string]string,
	provisioner string,
	params map[string]string) *StorageClass {

	return &StorageClass{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
		labels:      labels,
		client:      c,
		provisioner: provisioner,
		params:      params,
	}
}

// Get returns an existing StorageClass and an error if any.
func (s StorageClass) Get() (*storagev1.StorageClass, error) {
	sc := &storagev1.StorageClass{}
	err := s.client.Get(context.TODO(), s.NamespacedName, sc)
	return sc, err
}

// Delete deletes a StorageClass resource.
func (s StorageClass) Delete() error {
	return Delete(s.client, getStorageClass(s.Name, s.labels))
}

// Create creates a StorageClass resource.
func (s StorageClass) Create() error {
	sc := getStorageClass(s.Name, s.labels)
	sc.Provisioner = s.provisioner
	sc.Parameters = s.params
	return CreateOrUpdate(s.client, sc)
}

// getStorageClass returns a generic StorageClass object with the given name.
func getStorageClass(name string, labels map[string]string) *storagev1.StorageClass {
	return &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIstoragev1,
			Kind:       StorageClassKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
}
