package storageos

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	storageosapi "github.com/storageos/cluster-operator/internal/pkg/storageos"
	"github.com/storageos/cluster-operator/internal/pkg/storageos/mocks"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/go-api/v2"
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
		csiv1      bool
		wantbase   map[string]string
		wantcustom map[string]string
	}{
		{
			name:     "defaults",
			spec:     defaultSpec,
			csiv1:    true,
			wantbase: defaultConfig,
		},
		{
			name: "csi",
			spec: storageosv1.StorageOSClusterSpec{
				CSI: storageosv1.StorageOSClusterCSI{
					Enable: true,
				},
			},
			csiv1:    true,
			wantbase: defaultConfig,
		},
		{
			name: "csi v0 - override to csi v1",
			spec: storageosv1.StorageOSClusterSpec{
				CSI: storageosv1.StorageOSClusterCSI{
					Enable: true,
				},
			},
			csiv1:    false,
			wantbase: defaultConfig,
			wantcustom: map[string]string{
				csiEndpointEnvVar: "unix:///var/lib/kubelet/plugins_registry/storageos/csi.sock",
				csiVersionEnvVar:  "v1",
			},
		},
		{
			name: "shared-dir",
			spec: storageosv1.StorageOSClusterSpec{
				SharedDir: "some-dir-path",
			},
			csiv1:    true,
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
			csiv1:    true,
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
		// 	csiv1:    true,
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
			csiv1:    true,
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
			csiv1:    true,
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
			csiv1:    true,
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
			csiv1:    true,
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
			csiv1:    true,
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
			csiv1:    true,
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

			if got := configFromSpec(tt.spec, tt.csiv1); !reflect.DeepEqual(got, want) {
				t.Errorf("configFromSpec() got:\n%v\n want:\n%v\n", got, want)
			}
		})
	}
}

type mockReadCloser struct{}

func (m mockReadCloser) Read(p []byte) (n int, err error) { return 0, nil }
func (m mockReadCloser) Close() error                     { return nil }

func TestDeployment_ensureConfigMap(t *testing.T) {
	cmName := "storageos-node-config"
	cmNamespace := "kube-system"
	genCM := func(data map[string]string) *corev1.ConfigMap {
		return &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      cmName,
				Namespace: cmNamespace,
			},
			Data: data,
		}
	}

	patchCM := func(cm *corev1.ConfigMap, data map[string]string) *corev1.ConfigMap {
		new := cm.DeepCopy()
		for k, v := range data {
			new.Data[k] = v
		}
		return new
	}

	defaultCluster := api.Cluster{
		Id:                    uuid.New().String(),
		DisableTelemetry:      false,
		DisableCrashReporting: false,
		DisableVersionCheck:   false,
		LogLevel:              "info",
		LogFormat:             "json",
	}
	defaultConfig := configFromSpec(storageosv1.StorageOSClusterSpec{}, true)

	tests := []struct {
		name       string
		spec       storageosv1.StorageOSClusterSpec
		existing   *corev1.ConfigMap
		want       *corev1.ConfigMap
		prepare    func(m *mocks.MockControlPlane)
		apiOffline bool
		wantErr    bool
	}{
		{
			name:     "no update",
			existing: genCM(defaultConfig),
			want:     genCM(defaultConfig),
			prepare: func(m *mocks.MockControlPlane) {
				m.EXPECT().AuthenticateUser(gomock.Any(), gomock.Any()).Return(api.UserSession{}, &http.Response{
					Header: http.Header{
						"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
					},
					Body: mockReadCloser{},
				}, nil).Times(1)
				m.EXPECT().GetCluster(gomock.Any()).Return(defaultCluster, nil, nil).Times(1)
			},
		},
		{
			name: "bootstrap api online", // this won't happen in practice.
			want: genCM(defaultConfig),
			prepare: func(m *mocks.MockControlPlane) {
				m.EXPECT().AuthenticateUser(gomock.Any(), gomock.Any()).Return(api.UserSession{}, &http.Response{
					Header: http.Header{
						"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
					},
					Body: mockReadCloser{},
				}, nil).Times(1)
				m.EXPECT().GetCluster(gomock.Any()).Return(defaultCluster, nil, nil).Times(1)
			},
		},
		{
			name:       "bootstrap api offline", // default install.
			want:       genCM(defaultConfig),
			prepare:    func(m *mocks.MockControlPlane) {},
			apiOffline: true,
			wantErr:    false,
		},
		{
			name:       "cluster unhealthy",
			existing:   genCM(defaultConfig),
			want:       genCM(defaultConfig),
			prepare:    func(m *mocks.MockControlPlane) {},
			apiOffline: true,
			wantErr:    false,
		},
		{
			name:     "change while api down then recovered",
			existing: genCM(defaultConfig),
			want:     genCM(defaultConfig),
			prepare: func(m *mocks.MockControlPlane) {
				m.EXPECT().AuthenticateUser(gomock.Any(), gomock.Any()).Return(api.UserSession{}, &http.Response{
					Header: http.Header{
						"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
					},
					Body: mockReadCloser{},
				}, nil).Times(1)
				m.EXPECT().GetCluster(gomock.Any()).Return(api.Cluster{
					Id:                    uuid.New().String(),
					DisableTelemetry:      true,
					DisableCrashReporting: true,
					DisableVersionCheck:   true,
					LogLevel:              "warn",
					LogFormat:             "default",
				}, nil, nil).Times(1)
				m.EXPECT().UpdateCluster(gomock.Any(), api.UpdateClusterData{
					DisableTelemetry:      false,
					DisableCrashReporting: false,
					DisableVersionCheck:   false,
					LogLevel:              "info",
					LogFormat:             "json",
				}, gomock.Any()).Return(defaultCluster, nil, nil).Times(1)
			},
		},
		{
			name: "set debug",
			spec: storageosv1.StorageOSClusterSpec{
				Debug: true,
			},
			existing: genCM(defaultConfig),
			want: patchCM(genCM(defaultConfig), map[string]string{
				"LOG_LEVEL": "debug",
			}),
			prepare: func(m *mocks.MockControlPlane) {
				m.EXPECT().AuthenticateUser(gomock.Any(), gomock.Any()).Return(api.UserSession{}, &http.Response{
					Header: http.Header{
						"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
					},
					Body: mockReadCloser{},
				}, nil).Times(1)
				m.EXPECT().GetCluster(gomock.Any()).Return(defaultCluster, nil, nil).Times(1)
				m.EXPECT().UpdateCluster(gomock.Any(), api.UpdateClusterData{
					DisableTelemetry:      false,
					DisableCrashReporting: false,
					DisableVersionCheck:   false,
					LogLevel:              "debug",
					LogFormat:             "json",
				}, gomock.Any()).Return(defaultCluster, nil, nil).Times(1)
			},
		},
		{
			name: "disable telemetry",
			spec: storageosv1.StorageOSClusterSpec{
				DisableTelemetry: true,
			},
			existing: genCM(defaultConfig),
			want: patchCM(genCM(defaultConfig), map[string]string{
				"DISABLE_TELEMETRY":       "true",
				"DISABLE_CRASH_REPORTING": "true",
				"DISABLE_VERSION_CHECK":   "true",
			}),
			prepare: func(m *mocks.MockControlPlane) {
				m.EXPECT().AuthenticateUser(gomock.Any(), gomock.Any()).Return(api.UserSession{}, &http.Response{
					Header: http.Header{
						"Authorization": []string{"Bearer aaaabbbbcccdddeeeff"},
					},
					Body: mockReadCloser{},
				}, nil).Times(1)
				m.EXPECT().GetCluster(gomock.Any()).Return(defaultCluster, nil, nil).Times(1)
				m.EXPECT().UpdateCluster(gomock.Any(), api.UpdateClusterData{
					DisableTelemetry:      true,
					DisableCrashReporting: true,
					DisableVersionCheck:   true,
					LogLevel:              "info",
					LogFormat:             "json",
				}, gomock.Any()).Return(defaultCluster, nil, nil).Times(1)
			},
		},
	}
	for _, tt := range tests {
		var tt = tt
		t.Run(tt.name, func(t *testing.T) {
			mockCtrl := gomock.NewController(t)
			defer mockCtrl.Finish()
			mockCP := mocks.NewMockControlPlane(mockCtrl)

			c, deploy, err := setupFakeDeploymentWithAPI(storageosapi.Mocked(mockCP))
			if err != nil {
				t.Fatalf("failed to create deployment: %v", err)
			}

			ctx := context.Background()

			if tt.existing != nil {
				if err := c.Create(ctx, tt.existing); err != nil {
					t.Fatalf("failed to create existing configmap: %v", err)
				}
			}

			if tt.prepare != nil {
				tt.prepare(mockCP)
			}

			// Unhealthy if Join is not empty and host not avavilable.
			if tt.apiOffline {
				tt.spec.Join = "singlenode"
			}

			deploy.stos.Spec = tt.spec

			if err := deploy.ensureConfigMap(); (err != nil) != tt.wantErr {
				t.Errorf("Deployment.ensureConfigMap() error = %v, wantErr %v", err, tt.wantErr)
			}

			got := &corev1.ConfigMap{}
			if err := c.Get(ctx, types.NamespacedName{Name: cmName, Namespace: cmNamespace}, got); err != nil {
				t.Fatalf("failed to get updated configmap: %v", err)
			}

			if !reflect.DeepEqual(got.Data, tt.want.Data) {
				t.Errorf("Deployment.ensureConfigMap() \ngot  = %v\nwant = %v", got.Data, tt.want.Data)
			}
		})
	}
}
