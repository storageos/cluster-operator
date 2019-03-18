package storageos

import (
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func (s *Deployment) createIngress() error {
	ingress := &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-ingress",
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
			Annotations: s.stos.Spec.Ingress.Annotations,
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: s.stos.Spec.GetServiceName(),
				ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(s.stos.Spec.GetServiceExternalPort())},
			},
		},
	}

	if s.stos.Spec.Ingress.TLS {
		ingress.Spec.TLS = []v1beta1.IngressTLS{
			v1beta1.IngressTLS{
				Hosts:      []string{s.stos.Spec.Ingress.Hostname},
				SecretName: tlsSecretName,
			},
		}
	}

	return s.createOrUpdateObject(ingress)
}

func (s *Deployment) deleteIngress(name string) error {
	return s.deleteObject(s.getIngress(name))
}

func (s *Deployment) getIngress(name string) *v1beta1.Ingress {
	return &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": appName,
			},
			Annotations: s.stos.Spec.Ingress.Annotations,
		},
	}
}
