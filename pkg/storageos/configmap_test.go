package storageos

import (
	"os"
	"reflect"
	"testing"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

func Test_configFromSpec(t *testing.T) {

	v1DefaultSpec := storageosv1.StorageOSClusterSpec{}
	v1DefaultConfig := map[string]string{
		joinEnvVar:                 "",
		disableFencingEnvVar:       "false",
		disableTelemetryEnvVar:     "false",
		k8sSchedulerExtenderEnvVar: "true",
		v1NamespaceEnvVar:          "kube-system",
		logFormatEnvVar:            "text",
		logLevelEnvVar:             "info",
	}

	v2DefaultSpec := storageosv1.StorageOSClusterSpec{}
	v2DefaultConfig := map[string]string{
		csiEndpointEnvVar:           "unix:///var/lib/kubelet/plugins_registry/storageos/csi.sock",
		csiVersionEnvVar:            "v1",
		disableCrashReportingEnvVar: "false",
		disableTelemetryEnvVar:      "false",
		disableVersionCheckEnvVar:   "false",
		etcdEndpointsEnvVar:         "",
		k8sSchedulerExtenderEnvVar:  "true",
		v2NamespaceEnvVar:           "kube-system",
		logFormatEnvVar:             "json",
		logLevelEnvVar:              "info",
		// disableFencingEnvVar:        "false",
	}

	tests := []struct {
		name       string
		spec       storageosv1.StorageOSClusterSpec
		env        map[string]string
		csiv1      bool
		nodev2     bool
		wantbase   map[string]string
		wantcustom map[string]string
	}{
		{
			name:     "v1 defaults",
			spec:     v1DefaultSpec,
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
		},
		{
			name: "v1 csi",
			spec: storageosv1.StorageOSClusterSpec{
				CSI: storageosv1.StorageOSClusterCSI{
					Enable: true,
				},
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				csiEndpointEnvVar: "unix:///var/lib/kubelet/plugins_registry/storageos/csi.sock",
				csiVersionEnvVar:  "v1",
			},
		},
		{
			name: "v1 csi v0",
			spec: storageosv1.StorageOSClusterSpec{
				CSI: storageosv1.StorageOSClusterCSI{
					Enable: true,
				},
			},
			csiv1:    false,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				csiEndpointEnvVar: "unix:///var/lib/kubelet/plugins/storageos/csi.sock",
				csiVersionEnvVar:  "v0",
			},
		},
		{
			name: "v1 join",
			spec: storageosv1.StorageOSClusterSpec{
				Join: "1.2.3.4,5.6.7.8,4.3.2.1",
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				joinEnvVar: "1.2.3.4,5.6.7.8,4.3.2.1",
			},
		},
		{
			name: "v1 shared-dir",
			spec: storageosv1.StorageOSClusterSpec{
				SharedDir: "some-dir-path",
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				deviceDirEnvVar: "some-dir-path/devices",
			},
		},
		{
			name: "v1 disable telemetry",
			spec: storageosv1.StorageOSClusterSpec{
				DisableTelemetry: true,
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				disableTelemetryEnvVar: "true",
			},
		},
		{
			name: "v1 disable fencing",
			spec: storageosv1.StorageOSClusterSpec{
				DisableFencing: true,
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				disableFencingEnvVar: "true",
			},
		},
		{
			name: "v1 disable tcmu",
			spec: storageosv1.StorageOSClusterSpec{
				DisableTCMU: true,
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				disableTCMUEnvVar: "true",
			},
		},
		{
			name: "v1 force tcmu",
			spec: storageosv1.StorageOSClusterSpec{
				ForceTCMU: true,
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				forceTCMUEnvVar: "true",
			},
		},
		{
			name: "v1 kv backend only",
			spec: storageosv1.StorageOSClusterSpec{
				KVBackend: storageosv1.StorageOSClusterKVBackend{
					Backend: "etcd",
				},
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				kvBackendEnvVar: "etcd",
			},
		},
		{
			name: "v1 kv address only",
			spec: storageosv1.StorageOSClusterSpec{
				KVBackend: storageosv1.StorageOSClusterKVBackend{
					Address: "etcd-client:2379",
				},
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				kvAddrEnvVar: "etcd-client:2379",
			},
		},
		{
			name: "v1 external etcd",
			spec: storageosv1.StorageOSClusterSpec{
				KVBackend: storageosv1.StorageOSClusterKVBackend{
					Backend: "etcd",
					Address: "etcd-client:2379",
				},
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				kvBackendEnvVar: "etcd",
				kvAddrEnvVar:    "etcd-client:2379",
			},
		},
		{
			name: "v1 etcd TLS",
			spec: storageosv1.StorageOSClusterSpec{
				TLSEtcdSecretRefName:      "etcd-certs",
				TLSEtcdSecretRefNamespace: "default",
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				v1EtcdTLSClientCAEnvVar:   "/run/storageos/pki/etcd-client-ca.crt",
				v1EtcdTLSClientKeyEnvVar:  "/run/storageos/pki/etcd-client.key",
				v1EtcdTLSClientCertEnvVar: "/run/storageos/pki/etcd-client.crt",
			},
		},
		{
			name: "v1 external etcd with TLS",
			spec: storageosv1.StorageOSClusterSpec{
				KVBackend: storageosv1.StorageOSClusterKVBackend{
					Backend: "etcd",
					Address: "etcd-client:2379",
				},
				TLSEtcdSecretRefName:      "etcd-certs",
				TLSEtcdSecretRefNamespace: "default",
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				kvBackendEnvVar:           "etcd",
				kvAddrEnvVar:              "etcd-client:2379",
				v1EtcdTLSClientCAEnvVar:   "/run/storageos/pki/etcd-client-ca.crt",
				v1EtcdTLSClientKeyEnvVar:  "/run/storageos/pki/etcd-client.key",
				v1EtcdTLSClientCertEnvVar: "/run/storageos/pki/etcd-client.crt",
			},
		},
		{
			name: "v1 distro",
			spec: storageosv1.StorageOSClusterSpec{
				K8sDistro: "some-distro-name",
			},
			csiv1:    true,
			nodev2:   false,
			wantbase: v1DefaultConfig,
			wantcustom: map[string]string{
				k8sDistroEnvVar: "some-distro-name",
			},
		},

		{
			name:     "v2 defaults",
			spec:     v2DefaultSpec,
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
		},
		{
			name: "v2 csi",
			spec: storageosv1.StorageOSClusterSpec{
				CSI: storageosv1.StorageOSClusterCSI{
					Enable: true,
				},
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
		},
		{
			name: "v2 csi v0 - override to csi v1",
			spec: storageosv1.StorageOSClusterSpec{
				CSI: storageosv1.StorageOSClusterCSI{
					Enable: true,
				},
			},
			csiv1:    false,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				csiEndpointEnvVar: "unix:///var/lib/kubelet/plugins_registry/storageos/csi.sock",
				csiVersionEnvVar:  "v1",
			},
		},
		{
			name: "v2 shared-dir",
			spec: storageosv1.StorageOSClusterSpec{
				SharedDir: "some-dir-path",
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				deviceDirEnvVar: "some-dir-path/devices",
			},
		},
		{
			name: "v2 disable telemetry",
			spec: storageosv1.StorageOSClusterSpec{
				DisableTelemetry: true,
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				disableTelemetryEnvVar:      "true",
				disableCrashReportingEnvVar: "true",
				disableVersionCheckEnvVar:   "true",
			},
		},
		// Enable this when fencing is supported.
		// {
		// 	name: "v2 disable fencing",
		// 	spec: storageosv1.StorageOSClusterSpec{
		// 		DisableFencing: true,
		// 	},
		// 	csiv1:    true,
		// 	nodev2:   true,
		// 	wantbase: v2DefaultConfig,
		// 	wantcustom: map[string]string{
		// 		disableFencingEnvVar: "true",
		// 	},
		// },
		{
			name: "v2 disable tcmu",
			spec: storageosv1.StorageOSClusterSpec{
				DisableTCMU: true,
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				disableTCMUEnvVar: "true",
			},
		},
		{
			name: "v2 force tcmu",
			spec: storageosv1.StorageOSClusterSpec{
				ForceTCMU: true,
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				forceTCMUEnvVar: "true",
			},
		},
		{
			name: "v2 etcd TLS",
			spec: storageosv1.StorageOSClusterSpec{
				TLSEtcdSecretRefName:      "etcd-certs",
				TLSEtcdSecretRefNamespace: "default",
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				v2EtcdTLSClientCAEnvVar:   "/run/storageos/pki/etcd-client-ca.crt",
				v2EtcdTLSClientKeyEnvVar:  "/run/storageos/pki/etcd-client.key",
				v2EtcdTLSClientCertEnvVar: "/run/storageos/pki/etcd-client.crt",
			},
		},
		{
			name: "v2 distro",
			spec: storageosv1.StorageOSClusterSpec{
				K8sDistro: "some-distro-name",
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				k8sDistroEnvVar: "some-distro-name",
			},
		},
		{
			name: "v2 jaeger endpoint",
			spec: v2DefaultSpec,
			env: map[string]string{
				jaegerEndpointEnvVar: "http:/1.2.3.4:1234",
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				jaegerEndpointEnvVar: "http:/1.2.3.4:1234",
			},
		},
		{
			name: "v2 jaeger service name",
			spec: v2DefaultSpec,
			env: map[string]string{
				jaegerServiceNameEnvVar: "test-1234",
			},
			csiv1:    true,
			nodev2:   true,
			wantbase: v2DefaultConfig,
			wantcustom: map[string]string{
				jaegerServiceNameEnvVar: "test-1234",
			},
		},
	}
	for _, tt := range tests {
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

			if got := configFromSpec(tt.spec, tt.csiv1, tt.nodev2); !reflect.DeepEqual(got, want) {
				t.Errorf("configFromSpec() got:\n%v\n want:\n%v\n", got, want)
			}
		})
	}
}
