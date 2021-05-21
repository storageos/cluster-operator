package resource

import (
	"context"

	storagev1beta1 "k8s.io/api/storage/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CSIDriverKind is the name of the k8s CSIDriver resource kind.
const CSIDriverKind = "CSIDriver"

// CSIDriver implements k8s.Resource interface for k8s CSIDriver resource.
type CSIDriver struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	spec   *storagev1beta1.CSIDriverSpec
}

// NewCSIDriver returns an initialized CSIDriver.
func NewCSIDriver(
	c client.Client,
	name string,
	labels map[string]string,
	spec *storagev1beta1.CSIDriverSpec) *CSIDriver {
	return &CSIDriver{
		NamespacedName: types.NamespacedName{
			Name: name,
		},
		labels: labels,
		client: c,
		spec:   spec,
	}
}

// Get returns an existing CSIDriver and an error if any.
func (c CSIDriver) Get() (*storagev1beta1.CSIDriver, error) {
	csiDriver := &storagev1beta1.CSIDriver{}
	err := c.client.Get(context.TODO(), c.NamespacedName, csiDriver)
	return csiDriver, err
}

// Create creates a CSIDriver.
func (c CSIDriver) Create() error {
	csiDriver := getCSIDriver(c.Name, c.labels)
	csiDriver.Spec = *c.spec
	return Create(c.client, csiDriver)
}

// Delete deletes a CSIDriver.
func (c CSIDriver) Delete() error {
	return Delete(c.client, getCSIDriver(c.Name, c.labels))
}

// getCSIDriver returns a generic CSIDriver object.
func getCSIDriver(name string, labels map[string]string) *storagev1beta1.CSIDriver {
	return &storagev1beta1.CSIDriver{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIstoragev1beta1,
			Kind:       CSIDriverKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
}
