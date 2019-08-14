package storageos

import (
	"github.com/storageos/cluster-operator/pkg/util"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ingressName = "storageos-ingress"
)

func (s *Deployment) createIngress() error {
	spec := v1beta1.IngressSpec{
		Backend: &v1beta1.IngressBackend{
			ServiceName: s.stos.Spec.GetServiceName(),
			ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(s.stos.Spec.GetServiceExternalPort())},
		},
	}

	if s.stos.Spec.Ingress.TLS {
		spec.TLS = []v1beta1.IngressTLS{
			v1beta1.IngressTLS{
				Hosts:      []string{s.stos.Spec.Ingress.Hostname},
				SecretName: tlsSecretName,
			},
		}
	}

	return util.CreateIngress(s.client, ingressName, s.stos.Spec.GetResourceNS(), s.stos.Spec.Ingress.Annotations, spec)
}
