package storageos

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/storageos/cluster-operator/pkg/util/k8s"
)

// createService creates a service for storageos app with a label selector
// "kind" and value "daemonset" in order to select only the storageos node pods
// under the service. Any other value for "kind" will not be included in the
// service.
func (s *Deployment) createService() error {
	spec := &corev1.ServiceSpec{
		Type: corev1.ServiceType(s.stos.Spec.GetServiceType()),
		Ports: []corev1.ServicePort{
			{
				Name:       s.stos.Spec.GetServiceName(),
				Protocol:   "TCP",
				Port:       int32(s.stos.Spec.GetServiceInternalPort()),
				TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(s.stos.Spec.GetServiceExternalPort())},
			},
		},
		Selector: map[string]string{
			"app":  appName,
			"kind": daemonsetKind,
		},
	}

	// Add ServiceFor labels for the Service.
	labels := map[string]string{
		k8s.ServiceFor: "storageos-api-server",
	}

	if err := s.k8sResourceManager.Service(s.stos.Spec.GetServiceName(), s.stos.Spec.GetResourceNS(), labels, s.stos.Spec.Service.Annotations, spec).Create(); err != nil {
		return err
	}

	// Patch storageos-api secret with above service IP in apiAddress.
	if !s.stos.Spec.CSI.Enable {
		secret, err := s.k8sResourceManager.Secret(s.stos.Spec.SecretRefName, s.stos.Spec.SecretRefNamespace, nil, corev1.SecretTypeOpaque, nil).Get()
		if err != nil {
			return err
		}

		svc, err := s.k8sResourceManager.Service(s.stos.Spec.GetServiceName(), s.stos.Spec.GetResourceNS(), nil, nil, nil).Get()
		if err != nil {
			return err
		}

		apiAddress := fmt.Sprintf("tcp://%s:5705", svc.Spec.ClusterIP)
		secret.Data[apiAddressKey] = []byte(apiAddress)

		if err := s.client.Update(context.Background(), secret); err != nil {
			return err
		}
	}

	return nil
}

// APIServiceEndpoint returns the API's service endpoint suitable for use within
// the cluster.
func (s *Deployment) APIServiceEndpoint() string {
	return fmt.Sprintf("http://%s.%s.svc:%d", s.stos.Spec.GetServiceName(), s.stos.Spec.GetResourceNS(), s.stos.Spec.GetServiceExternalPort())
}
