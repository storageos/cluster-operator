package storageos

import (
	"context"
	"fmt"

	"github.com/storageos/cluster-operator/pkg/util"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// createService creates a service for storageos app with a label selector
// "kind" and value "daemonset" in order to select only the storageos node pods
// under the service. Any other value for "kind" will not be included in the
// service.
func (s *Deployment) createService() error {
	spec := corev1.ServiceSpec{
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

	if err := util.CreateService(s.client, s.stos.Spec.GetServiceName(), s.stos.Spec.GetResourceNS(), s.stos.Spec.Service.Annotations, spec); err != nil {
		return err
	}

	// Patch storageos-api secret with above service IP in apiAddress.
	if !s.stos.Spec.CSI.Enable {
		secret := &corev1.Secret{}
		nsNameSecret := types.NamespacedName{
			Namespace: s.stos.Spec.SecretRefNamespace,
			Name:      s.stos.Spec.SecretRefName,
		}
		if err := s.client.Get(context.Background(), nsNameSecret, secret); err != nil {
			return err
		}

		svc := &corev1.Service{}
		nsNameService := types.NamespacedName{
			Namespace: s.stos.Spec.GetResourceNS(),
			Name:      s.stos.Spec.GetServiceName(),
		}
		if err := s.client.Get(context.Background(), nsNameService, svc); err != nil {
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
