package toleration

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestDeepEqual(t *testing.T) {
	var secondsA int64 = 30
	var secondsB int64 = 40
	tests := []struct {
		name string
		a    []corev1.Toleration
		b    []corev1.Toleration
		want bool
	}{
		{
			name: "both empty",
			a:    []corev1.Toleration{},
			b:    []corev1.Toleration{},
			want: true,
		},
		{
			name: "one empty",
			a:    []corev1.Toleration{},
			b: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			want: false,
		},
		{
			name: "other empty",
			a: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			b:    []corev1.Toleration{},
			want: false,
		},
		{
			name: "same",
			a: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			b: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			want: true,
		},
		{
			name: "different seconds",
			a: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			b: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsB,
				},
			},
			want: false,
		},
		{
			name: "seconds nil",
			a: []corev1.Toleration{
				{
					Key:      TaintNodeNotReady,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoExecute,
				},
			},
			b: []corev1.Toleration{
				{
					Key:      TaintNodeNotReady,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoExecute,
				},
			},
			want: true,
		},
		{
			name: "different seconds one nil",
			a: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			b: []corev1.Toleration{
				{
					Key:      TaintNodeNotReady,
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoExecute,
				},
			},
			want: false,
		},
		{
			name: "different key",
			a: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			b: []corev1.Toleration{
				{
					Key:               TaintNodeUnreachable,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			want: false,
		},
		{
			name: "different operator",
			a: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			b: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpEqual,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			want: false,
		},
		{
			name: "different effect",
			a: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
				},
			},
			b: []corev1.Toleration{
				{
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoSchedule,
					TolerationSeconds: &secondsA,
				},
			},
			want: false,
		},
		{
			name: "multiple same",
			a: []corev1.Toleration{
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
			},
			b: []corev1.Toleration{
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
			},
			want: true,
		},
		{
			name: "multiple different",
			a: []corev1.Toleration{
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
			},
			b: []corev1.Toleration{
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
					Key:               TaintNodeNotReady,
					Operator:          corev1.TolerationOpExists,
					Effect:            corev1.TaintEffectNoExecute,
					TolerationSeconds: &secondsA,
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
			},
			want: false,
		},
	}
	for _, tt := range tests {
		var tt = tt
		t.Run(tt.name, func(t *testing.T) {
			if got := DeepEqual(tt.a, tt.b); got != tt.want {
				t.Errorf("DeepEqual() = %t, want %t", got, tt.want)
			}
		})
	}
}
