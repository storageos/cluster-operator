package storageos

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// createService creates a service for storageos app with a label selector
// "kind" and value "daemonset" in order to select only the storageos node pods
// under the service. Any other value for "kind" will not be included in the
// service.
func (s *Deployment) createService() error {
	svc := s.getService(s.stos.Spec.GetServiceName())
	svc.Spec = corev1.ServiceSpec{
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

	if err := s.client.Create(context.Background(), svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create %s: %v", svc.GroupVersionKind().Kind, err)
	}
	// if err := s.createOrUpdateObject(svc); err != nil {
	// 	return err
	// }

	// Patch storageos-api secret with above service IP in apiAddress.
	if !s.stos.Spec.CSI.Enable {
		secret := &corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.stos.Spec.SecretRefName,
				Namespace: s.stos.Spec.SecretRefNamespace,
			},
		}
		nsNameSecret := types.NamespacedName{
			Namespace: secret.ObjectMeta.GetNamespace(),
			Name:      secret.ObjectMeta.GetName(),
		}
		if err := s.client.Get(context.Background(), nsNameSecret, secret); err != nil {
			return err
		}

		nsNameService := types.NamespacedName{
			Namespace: svc.ObjectMeta.GetNamespace(),
			Name:      svc.ObjectMeta.GetName(),
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

func (s *Deployment) deleteService(name string) error {
	return s.deleteObject(s.getService(name))
}

func (s *Deployment) getService(name string) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
			Annotations: s.stos.Spec.Service.Annotations,
		},
	}
}
