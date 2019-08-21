package resource

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// StatefulSetKind is the name of the k8s StatefulSet resource kind.
const StatefulSetKind = "StatefulSet"

// StatefulSet implements k8s.Resource interface for k8s StatefulSet resource.
type StatefulSet struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	spec   *appsv1.StatefulSetSpec
}

// NewStatefulSet returns an initialized StatefulSet.
func NewStatefulSet(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	spec *appsv1.StatefulSetSpec) *StatefulSet {

	return &StatefulSet{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels: labels,
		client: c,
		spec:   spec,
	}
}

// Get returns an existing StatefulSet and an error if any.
func (s StatefulSet) Get() (*appsv1.StatefulSet, error) {
	ss := &appsv1.StatefulSet{}
	err := s.client.Get(context.TODO(), s.NamespacedName, ss)
	return ss, err
}

// Create creates a StatefulSet.
func (s StatefulSet) Create() error {
	statefulset := getStatefulSet(s.Name, s.Namespace, s.labels)
	statefulset.Spec = *s.spec
	return CreateOrUpdate(s.client, statefulset)
}

// Delete deletes a StatefulSet.
func (s StatefulSet) Delete() error {
	return Delete(s.client, getStatefulSet(s.Name, s.Namespace, s.labels))
}

// getStatefulSet returns a generic StatefulSet object.
func getStatefulSet(name, namespace string, labels map[string]string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIappsv1,
			Kind:       StatefulSetKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
