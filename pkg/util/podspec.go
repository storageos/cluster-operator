package util

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
)

// AddTolerations adds given tolerations to the given pod spec.
func AddTolerations(podSpec *corev1.PodSpec, tolerations []corev1.Toleration) error {
	for i := range tolerations {
		if tolerations[i].Operator == corev1.TolerationOpExists && tolerations[i].Value != "" {
			return fmt.Errorf("key(%s): toleration value must be empty when `operator` is 'Exists'", tolerations[i].Key)
		}
	}
	if len(tolerations) > 0 {
		podSpec.Tolerations = tolerations
	}
	return nil
}
