package resource

import (
	"context"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// DeploymentKind is the name of k8s Deployment resource kind.
const DeploymentKind = "Deployment"

// Deployment implements k8s.Resource interface for k8s Deployment resource.
type Deployment struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	spec   *appsv1.DeploymentSpec
}

// NewDeployment returns an initialized Deployment.
func NewDeployment(
	c client.Client,
	name, namespace string,
	labels map[string]string,
	spec *appsv1.DeploymentSpec) *Deployment {
	return &Deployment{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels: labels,
		client: c,
		spec:   spec,
	}
}

// Get returns an existing Deployment and an error if any.
func (d Deployment) Get() (*appsv1.Deployment, error) {
	dep := &appsv1.Deployment{}
	err := d.client.Get(context.TODO(), d.NamespacedName, dep)
	return dep, err
}

// Create creates a new Deployment resource.
func (d Deployment) Create() error {
	deployment := getDeployment(d.Name, d.Namespace, d.labels)
	deployment.Spec = *d.spec
	return CreateOrUpdate(d.client, deployment)
}

// Delete deletes an existing Deployment resource.
func (d Deployment) Delete() error {
	return Delete(d.client, getDeployment(d.Name, d.Namespace, d.labels))
}

// getDeployment returns a generic Deployment object given the name and
// namespace.
func getDeployment(name, namespace string, labels map[string]string) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIappsv1,
			Kind:       DeploymentKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
