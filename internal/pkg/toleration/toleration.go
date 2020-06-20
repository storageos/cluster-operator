package toleration

import (
	corev1 "k8s.io/api/core/v1"
)

// Taint constants taken from k8s.io/api release-1.17 branch.
// https://github.com/kubernetes/api/blob/release-1.17/core/v1/well_known_taints.go
// TODO: Replace these constants with k8s.io/api constants after updating the
// k8s dependency. TaintNodeOutOfDisk is not an upstream constant.
const (
	// TaintNodeNotReady will be added when node is not ready
	// and feature-gate for TaintBasedEvictions flag is enabled,
	// and removed when node becomes ready.
	TaintNodeNotReady = "node.kubernetes.io/not-ready"

	// TaintNodeUnreachable will be added when node becomes unreachable
	// (corresponding to NodeReady status ConditionUnknown)
	// and feature-gate for TaintBasedEvictions flag is enabled,
	// and removed when node becomes reachable (NodeReady status ConditionTrue).
	TaintNodeUnreachable = "node.kubernetes.io/unreachable"

	// TaintNodeUnschedulable will be added when node becomes unschedulable
	// and feature-gate for TaintNodesByCondition flag is enabled,
	// and removed when node becomes scheduable.
	TaintNodeUnschedulable = "node.kubernetes.io/unschedulable"

	// TaintNodeMemoryPressure will be added when node has memory pressure
	// and feature-gate for TaintNodesByCondition flag is enabled,
	// and removed when node has enough memory.
	TaintNodeMemoryPressure = "node.kubernetes.io/memory-pressure"

	// TaintNodeDiskPressure will be added when node has disk pressure
	// and feature-gate for TaintNodesByCondition flag is enabled,
	// and removed when node has enough disk.
	TaintNodeDiskPressure = "node.kubernetes.io/disk-pressure"

	// TaintNodeNetworkUnavailable will be added when node's network is unavailable
	// and feature-gate for TaintNodesByCondition flag is enabled,
	// and removed when network becomes ready.
	TaintNodeNetworkUnavailable = "node.kubernetes.io/network-unavailable"

	// TaintNodePIDPressure will be added when node has pid pressure
	// and feature-gate for TaintNodesByCondition flag is enabled,
	// and removed when node has enough disk.
	TaintNodePIDPressure = "node.kubernetes.io/pid-pressure"

	// TaintNodePIDPressure will be added when node runs out of disk space, and
	// removed when disk space becomes available.
	TaintNodeOutOfDisk = "node.kubernetes.io/out-of-disk"
)

// GetDefaultTolerations returns a collection of default tolerations for
// StorageOS related resources.
// NOTE: An empty effect matches all effects with the given key.
func GetDefaultTolerations() []corev1.Toleration {
	return []corev1.Toleration{
		{
			Key:      TaintNodeDiskPressure,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodeMemoryPressure,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodeNetworkUnavailable,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodeNotReady,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodeOutOfDisk,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodePIDPressure,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodeUnreachable,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
		{
			Key:      TaintNodeUnschedulable,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
	}
}
