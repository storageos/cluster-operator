package nfs

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/storageos/cluster-operator/pkg/util/k8s"
)

func (d *Deployment) ensureService(nfsPort int) error {
	// If no error in getting the service, service already exists, do nothing.
	if _, err := d.getServerService(); err == nil {
		return nil
	}

	labels := map[string]string{
		k8s.ServiceFor: "nfs-server",
	}

	// Couldn't get any existing service. Create a new service.
	if err := d.createService(d.nfsServer.Name, NFSPortName, nfsPort, labels); err != nil {
		return err
	}
	return nil
}

// createMetricsService creates a Service for metrics at the given port number.
func (d *Deployment) createMetricsService(metricsPort int) error {
	labels := map[string]string{
		k8s.ServiceFor: "nfs-metrics",
	}
	if err := d.createService(d.getMetricsServiceName(), MetricsPortName, metricsPort, labels); err != nil {
		return err
	}
	return nil
}

func (d *Deployment) createService(name string, portName string, port int, labels map[string]string) error {
	spec := &corev1.ServiceSpec{
		Selector: d.labelsForStatefulSet(),
		Type:     corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{
				Name:       portName,
				Port:       int32(port),
				TargetPort: intstr.FromInt(port),
			},
		},
	}
	return d.k8sResourceManager.Service(name, d.nfsServer.Namespace, labels, nil, spec).Create()
}

// getServerService returns the NFS Server endpoint service.
func (d *Deployment) getServerService() (*corev1.Service, error) {
	return d.k8sResourceManager.Service(d.nfsServer.Name, d.nfsServer.Namespace, nil, nil, nil).Get()
}

// getMetricsServiceName returns the name of the metrics service.
func (d *Deployment) getMetricsServiceName() string {
	return fmt.Sprintf("%s-%s", d.nfsServer.Name, MetricsPortName)
}

// getMetricsService returns the NFS Server metrics endpoint service.
func (d *Deployment) getMetricsService() (*corev1.Service, error) {
	return d.k8sResourceManager.Service(d.getMetricsServiceName(), d.nfsServer.Namespace, nil, nil, nil).Get()
}
