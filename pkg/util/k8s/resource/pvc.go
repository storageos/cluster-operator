package resource

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PVCKind is the name of the k8s PersistentVolumeClaim resource kind.
const PVCKind = "PersistentVolumeClaim"

// PVC implements k8s.Resource interface for k8s PersistentVolumeClaim resource.
type PVC struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	spec   *corev1.PersistentVolumeClaimSpec
}

// NewPVC returns an initialized PVC.
func NewPVC(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	spec *corev1.PersistentVolumeClaimSpec) *PVC {

	return &PVC{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels: labels,
		client: c,
		spec:   spec,
	}
}

// Get returns an existing PVC and an error if any.
func (p PVC) Get() (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{}
	err := p.client.Get(context.TODO(), p.NamespacedName, pvc)
	return pvc, err
}

// Create creates a PVC.
func (p PVC) Create() error {
	pvc := getPVC(p.Name, p.Namespace, p.labels)
	pvc.Spec = *p.spec
	return CreateOrUpdate(p.client, pvc)
}

// Delete deletes a PVC.
func (p PVC) Delete() error {
	return Delete(p.client, getPVC(p.Name, p.Namespace, p.labels))
}

// getPVC returns a generic PersistentVolumeClaim object.
func getPVC(name, namespace string, labels map[string]string) *corev1.PersistentVolumeClaim {
	return &corev1.PersistentVolumeClaim{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIv1,
			Kind:       PVCKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
