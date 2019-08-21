package nfs

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (d *Deployment) ensureService(nfsPort int, metricsPort int) error {
	_, err := d.k8sResourceManager.Service(d.nfsServer.Name, d.nfsServer.Namespace, nil, nil).Get()
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
	spec := &corev1.ServiceSpec{
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

	return d.k8sResourceManager.Service(d.nfsServer.Name, d.nfsServer.Namespace, nil, spec).Create()
}
