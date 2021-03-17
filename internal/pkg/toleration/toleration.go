package toleration

import (
	"sort"

	corev1 "k8s.io/api/core/v1"
)

// Taint constants taken from k8s.io/api release-1.17 branch.
// https://github.com/kubernetes/api/blob/release-1.17/core/v1/well_known_taints.go
// TODO: Replace these constants with k8s.io/api constants after updating the
// k8s dependency. TaintNodeOutOfDisk is not an upstream constant.
const (
	// TaintNodeNotReady will be added when node is not ready
	// and removed when node becomes ready.
	TaintNodeNotReady = "node.kubernetes.io/not-ready"

	// TaintNodeUnreachable will be added when node becomes unreachable
	// (corresponding to NodeReady status ConditionUnknown)
	// and removed when node becomes reachable (NodeReady status ConditionTrue).
	TaintNodeUnreachable = "node.kubernetes.io/unreachable"

	// TaintNodeUnschedulable will be added when node becomes unschedulable
	// and removed when node becomes scheduable.
	TaintNodeUnschedulable = "node.kubernetes.io/unschedulable"

	// TaintNodeMemoryPressure will be added when node has memory pressure
	// and removed when node has enough memory.
	TaintNodeMemoryPressure = "node.kubernetes.io/memory-pressure"

	// TaintNodeDiskPressure will be added when node has disk pressure
	// and removed when node has enough disk.
	//
	// `"node.kubernetes.io/out-of-disk"` was removed in k8s 1.13.
	TaintNodeDiskPressure = "node.kubernetes.io/disk-pressure"

	// TaintNodeNetworkUnavailable will be added when node's network is unavailable
	// and removed when network becomes ready.
	TaintNodeNetworkUnavailable = "node.kubernetes.io/network-unavailable"

	// TaintNodePIDPressure will be added when node has pid pressure, and
	// removed when node pid usage has reduced below `/proc/sys/kernel/pid_max`.
	TaintNodePIDPressure = "node.kubernetes.io/pid-pressure"
)

// GetDefaultNodeTolerations returns a collection of tolerations suitable for
// StorageOS node related resources.
//
// Node resources should avoid being evicted.
//
// NOTE: An empty effect matches all effects with the given key.
func GetDefaultNodeTolerations() []corev1.Toleration {
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

// GetDefaultHelperTolerations returns a collection of tolerations suitable for
// StorageOS related helpers.
//
// Helpers should failover when a node is unresponsive but as they have minimal
// dependencies they are able to tolerate some taints (e.g. disk-pressure).
//
// For other taints, only tolerate them for the given period.
//
// NOTE: An empty effect matches all effects with the given key.
func GetDefaultHelperTolerations(tolerationSeconds int64) []corev1.Toleration {
	return []corev1.Toleration{
		{
			Key:               TaintNodeNotReady,
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: &tolerationSeconds,
		},
		{
			Key:               TaintNodeUnreachable,
			Operator:          corev1.TolerationOpExists,
			Effect:            corev1.TaintEffectNoExecute,
			TolerationSeconds: &tolerationSeconds,
		},
		{
			Key:      TaintNodeDiskPressure,
			Operator: corev1.TolerationOpExists,
			Effect:   "",
		},
	}
}

// DeepEqual compares two slices of tolerations for equality.
func DeepEqual(a []corev1.Toleration, b []corev1.Toleration) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[string]corev1.Toleration)
	for _, t := range a {
		aMap[t.Key] = t
	}
	bMap := make(map[string]corev1.Toleration)
	for _, t := range b {
		bMap[t.Key] = t
	}

	for k, aTol := range aMap {
		bTol, ok := bMap[k]
		if !ok {
			return false
		}
		if !aTol.MatchToleration(&bTol) {
			return false
		}
		// MatchTolerations does not compare TolerationSeconds.
		if aTol.TolerationSeconds == nil && bTol.TolerationSeconds == nil {
			continue
		}
		if aTol.TolerationSeconds == nil || bTol.TolerationSeconds == nil {
			return false
		}
		if *aTol.TolerationSeconds != *bTol.TolerationSeconds {
			return false
		}
	}
	return true
}

// Sort sorts a slice of tolerations.
func Sort(a []corev1.Toleration) {
	sort.SliceStable(a, func(i, j int) bool { return a[i].Key < a[j].Key })
}
