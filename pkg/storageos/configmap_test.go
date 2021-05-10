package storageos

import (
	"os"
	"reflect"
	"testing"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

func Test_configFromSpec(t *testing.T) {
	defaultSpec := storageosv1.StorageOSClusterSpec{}
	defaultConfig := map[string]string{
		csiEndpointEnvVar:           "unix:///var/lib/kubelet/plugins_registry/storageos/csi.sock",
		csiVersionEnvVar:            "v1",
		disableCrashReportingEnvVar: "false",
		disableTelemetryEnvVar:      "false",
		disableVersionCheckEnvVar:   "false",
		etcdEndpointsEnvVar:         "",
		k8sSchedulerExtenderEnvVar:  "true",
		namespaceEnvVar:             "kube-system",
		logFormatEnvVar:             "json",
		logLevelEnvVar:              "info",
		recommendedPidLimitEnvVar:   "32768",
		// disableFencingEnvVar:        "false",
	}

	tests := []struct {
		name       string
		spec       storageosv1.StorageOSClusterSpec
		env        map[string]string
		wantbase   map[string]string
		wantcustom map[string]string
	}{
		{
			name:     "defaults",
			spec:     defaultSpec,
			wantbase: defaultConfig,
		},
		{
			name: "csi",
			spec: storageosv1.StorageOSClusterSpec{
				CSI: storageosv1.StorageOSClusterCSI{
					Enable: true,
				},
			},
			wantbase: defaultConfig,
		},
		{
			name: "shared-dir",
			spec: storageosv1.StorageOSClusterSpec{
				SharedDir: "some-dir-path",
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				deviceDirEnvVar: "some-dir-path/devices",
			},
		},
		{
			name: "disable telemetry",
			spec: storageosv1.StorageOSClusterSpec{
				DisableTelemetry: true,
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				disableTelemetryEnvVar:      "true",
				disableCrashReportingEnvVar: "true",
				disableVersionCheckEnvVar:   "true",
			},
		},
		// Enable this when fencing is supported.
		// {
		// 	name: "disable fencing",
		// 	spec: storageosv1.StorageOSClusterSpec{
		// 		DisableFencing: true,
		// 	},
		// 	wantbase: v2DefaultConfig,
		// 	wantcustom: map[string]string{
		// 		disableFencingEnvVar: "true",
		// 	},
		// },
		{
			name: "disable tcmu",
			spec: storageosv1.StorageOSClusterSpec{
				DisableTCMU: true,
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				disableTCMUEnvVar: "true",
			},
		},
		{
			name: "force tcmu",
			spec: storageosv1.StorageOSClusterSpec{
				ForceTCMU: true,
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				forceTCMUEnvVar: "true",
			},
		},
		{
			name: "etcd TLS",
			spec: storageosv1.StorageOSClusterSpec{
				TLSEtcdSecretRefName:      "etcd-certs",
				TLSEtcdSecretRefNamespace: "default",
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				etcdTLSClientCAEnvVar:   "/run/storageos/pki/etcd-client-ca.crt",
				etcdTLSClientKeyEnvVar:  "/run/storageos/pki/etcd-client.key",
				etcdTLSClientCertEnvVar: "/run/storageos/pki/etcd-client.crt",
			},
		},
		{
			name: "distro",
			spec: storageosv1.StorageOSClusterSpec{
				K8sDistro: "some-distro-name",
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				k8sDistroEnvVar: "some-distro-name",
			},
		},
		{
			name: "jaeger endpoint",
			spec: defaultSpec,
			env: map[string]string{
				jaegerEndpointEnvVar: "http:/1.2.3.4:1234",
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				jaegerEndpointEnvVar: "http:/1.2.3.4:1234",
			},
		},
		{
			name: "jaeger service name",
			spec: defaultSpec,
			env: map[string]string{
				jaegerServiceNameEnvVar: "test-1234",
			},
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				jaegerServiceNameEnvVar: "test-1234",
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			// Set wanted env vars.
			// Don't parallelize tests as they will conflict.
			for k, v := range tt.env {
				os.Setenv(k, v)
				defer func(k string) {
					if err := os.Unsetenv(k); err != nil {
						t.Fatal(err)
					}
				}(k)
			}

			var want = make(map[string]string)
			for k, v := range tt.wantbase {
				want[k] = v
			}
			for k, v := range tt.wantcustom {
				want[k] = v
			}

			if got := configFromSpec(tt.spec); !reflect.DeepEqual(got, want) {
				t.Errorf("configFromSpec() got:\n%v\n want:\n%v\n", got, want)
			}
		})
	}
}
