package nfs

import (
	"context"

	"github.com/storageos/cluster-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (d *Deployment) ensureService(nfsPort int, metricsPort int) error {
	_, err := d.getService(d.nfsServer.Name, d.nfsServer.Namespace)
	// If no error in getting the service, service already exists, do nothing.
	if err == nil {
		return nil
	}
	// Couldn't get any existing service. Create a new service.
	if err := d.createService(nfsPort, metricsPort); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) createService(nfsPort int, metricsPort int) error {
	spec := corev1.ServiceSpec{
		Selector: d.labelsForStatefulSet(d.nfsServer.Name, map[string]string{}),
		Type:     corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{
				Name:       "nfs",
				Port:       int32(nfsPort),
				TargetPort: intstr.FromInt(int(nfsPort)),
			},
			{
				Name:       "metrics",
				Port:       int32(metricsPort),
				TargetPort: intstr.FromInt(int(metricsPort)),
			},
		},
	}

	return util.CreateService(d.client, d.nfsServer.Name, d.nfsServer.Namespace, nil, spec)
}

func (d *Deployment) getService(name string, namespace string) (*corev1.Service, error) {
	service := &corev1.Service{}

	namespacedService := types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
	if err := d.client.Get(context.TODO(), namespacedService, service); err != nil {
		return nil, err
	}
	return service, nil
}
