package resource

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigMapKind is the name of k8s ConfigMap resource kind.
const ConfigMapKind = "ConfigMap"

// ConfigMap implements k8s.Resource interface for k8s ConfigMap resource.
type ConfigMap struct {
	types.NamespacedName
	labels map[string]string
	client client.Client
	data   map[string]string
}

// NewConfigMap returns an initialized ConfigMap.
func NewConfigMap(c client.Client, name, namespace string, labels map[string]string, data map[string]string) *ConfigMap {
	return &ConfigMap{
		NamespacedName: types.NamespacedName{
			Name:      name,
			Namespace: namespace,
		},
		labels: labels,
		client: c,
		data:   data,
	}
}

// Get returns an existing ConfigMap and an error if any.
func (cm ConfigMap) Get() (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{}
	if err := cm.client.Get(context.TODO(), cm.NamespacedName, configmap); err != nil {
		return nil, err
	}
	return configmap, nil
}

// Create creates a new k8s ConfigMap resource.
func (cm ConfigMap) Create() error {
	configmap := getConfigMap(cm.Name, cm.Namespace, cm.labels)
	configmap.Data = cm.data
	return CreateOrUpdate(cm.client, configmap)
}

// Delete deletes an existing k8s resource.
func (cm ConfigMap) Delete() error {
	return Delete(cm.client, getConfigMap(cm.Name, cm.Namespace, cm.labels))
}

// getConfigMap returns an empty ConfigMap object. This can be used while
// creating a configmap resource.
func getConfigMap(name, namespace string, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: APIv1,
			Kind:       ConfigMapKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
	}
}
