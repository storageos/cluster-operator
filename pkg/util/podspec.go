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

// AddRequiredNodeAffinity adds required node affinity to the given pod spec.
func AddRequiredNodeAffinity(podSpec *corev1.PodSpec, terms []corev1.NodeSelectorTerm) {
	if len(terms) == 0 {
		return
	}
	podSpec.Affinity = &corev1.Affinity{
		NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: terms,
			},
		},
	}
}
