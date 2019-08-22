package resource

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DaemonSetKind is the name of k8s DaemonSet resource kind.
const DaemonSetKind = "DaemonSet"

// DaemonSet implements k8s.Resource interface for k8s DaemonSet resource.
type DaemonSet struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	spec   *appsv1.DaemonSetSpec
}

// NewDaemonSet returns an initialized DaemonSet.
func NewDaemonSet(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	spec *appsv1.DaemonSetSpec) *DaemonSet {

	return &DaemonSet{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels: labels,
		client: c,
		spec:   spec,
	}
}

// Get returns an existing DaemonSet and an error if any.
func (d DaemonSet) Get() (*appsv1.DaemonSet, error) {
	ds := &appsv1.DaemonSet{}
	err := d.client.Get(context.TODO(), d.NamespacedName, ds)
	return ds, err
}

// Create creates a new k8s DaemonSet resource.
func (d DaemonSet) Create() error {
	daemonset := getDaemonSet(d.Name, d.Namespace, d.labels)
	daemonset.Spec = *d.spec
	return CreateOrUpdate(d.client, daemonset)
}

// Delete deletes an existing DaemonSet resource.
func (d DaemonSet) Delete() error {
	return Delete(d.client, getDaemonSet(d.Name, d.Namespace, d.labels))
}

// getDaemonSet returns a generic DaemonSet object.
func getDaemonSet(name, namespace string, labels map[string]string) *appsv1.DaemonSet {
	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIappsv1,
			Kind:       DaemonSetKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
