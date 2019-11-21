package storageos

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	storageosapis "github.com/storageos/cluster-operator/pkg/apis"
	api "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

var gvk = schema.GroupVersionKind{
	Group:   "storageos.com",
	Version: "v1",
	Kind:    "StorageOSCluster",
}

var testScheme = runtime.NewScheme()

const defaultNS = "storageos"

func setupFakeDeployment() (client.Client, *Deployment) {
	c := fake.NewFakeClientWithScheme(testScheme)
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
	}

	deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
	return c, deploy
}

func testSetup() {
	// Register all the schemes.
	kscheme.AddToScheme(testScheme)
	apiextensionsv1beta1.AddToScheme(testScheme)
	storageosapis.AddToScheme(testScheme)
}

func TestMain(m *testing.M) {
	testSetup()
	os.Exit(m.Run())
}

func TestCreateNamespace(t *testing.T) {
	c, deploy := setupFakeDeployment()
	if err := deploy.createNamespace(); err != nil {
		t.Fatal("failed to create namespace", err)
	}

	// Fetch the created namespace and check if it's a child of StorageOSCluster.
	nsName := types.NamespacedName{Name: defaultNS}
	wantNS := &corev1.Namespace{}
	if err := c.Get(context.TODO(), nsName, wantNS); err != nil {
		t.Fatal("failed to get the created object", err)
	}
}

func TestCreateDaemonSet(t *testing.T) {
	clusterName := "my-stos-cluster"
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: defaultNS,
		},
	}

	// etcd secret containing TLS certs. This exists before storageos cluster
	// is created.
	etcdSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd-certs",
			Namespace: "default",
		},
		Data: map[string][]byte{
			tlsEtcdCA:         []byte("someetcdca"),
			tlsEtcdClientCert: []byte("someetcdclientcert"),
			tlsEtcdClientKey:  []byte("someetcdclientkey"),
		},
	}

	testcases := []struct {
		name                 string
		spec                 api.StorageOSClusterSpec
		wantEnableCSI        bool
		wantSharedDir        string
		wantDisableTelemetry bool
		wantDisableFencing   bool
		wantDisableTCMU      bool
		wantForceTCMU        bool
		wantTLSEtcd          bool
		wantK8sDistro        string
	}{
		{
			name: "legacy-daemonset",
			spec: api.StorageOSClusterSpec{
				SecretRefName:      "foo-secret",
				SecretRefNamespace: "default",
			},
		},
		{
			name: "csi-daemonset",
			spec: api.StorageOSClusterSpec{
				SecretRefName:      "foo-secret",
				SecretRefNamespace: "default",
				CSI: api.StorageOSClusterCSI{
					Enable: true,
				},
			},
			wantEnableCSI: true,
		},
		{
			name: "shared-dir",
			spec: api.StorageOSClusterSpec{
				SharedDir: "some-dir-path",
			},
			wantSharedDir: "some-dir-path",
		},
		{
			name: "disable telemetry",
			spec: api.StorageOSClusterSpec{
				DisableTelemetry: true,
			},
			wantDisableTelemetry: true,
		},
		{
			name: "disable fencing",
			spec: api.StorageOSClusterSpec{
				DisableFencing: true,
			},
			wantDisableFencing: true,
		},
		{
			name: "disable tcmu",
			spec: api.StorageOSClusterSpec{
				DisableTCMU: true,
			},
			wantDisableTCMU: true,
		},
		{
			name: "force tcmu",
			spec: api.StorageOSClusterSpec{
				ForceTCMU: true,
			},
			wantForceTCMU: true,
		},
		{
			name: "etcd TLS",
			spec: api.StorageOSClusterSpec{
				TLSEtcdSecretRefName:      "etcd-certs",
				TLSEtcdSecretRefNamespace: "default",
			},
			wantTLSEtcd: true,
		},
		{
			name: "distro",
			spec: api.StorageOSClusterSpec{
				K8sDistro: "some-distro-name",
			},
			wantK8sDistro: "some-distro-name",
		},
	}

	for _, tc := range testcases {
		// Create fake client with pre-existing resources.
		c := fake.NewFakeClientWithScheme(testScheme, etcdSecret)

		stosCluster.Spec = tc.spec
		deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
		if err := deploy.createDaemonSet(); err != nil {
			t.Fatal("failed to create daemonset", err)
		}

		nsName := types.NamespacedName{
			Name:      daemonsetName,
			Namespace: defaultNS,
		}
		createdDaemonset := &appsv1.DaemonSet{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "apps/v1",
				Kind:       "DaemonSet",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      daemonsetName,
				Namespace: defaultNS,
			},
		}
		if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
			t.Fatal("failed to get the created object", err)
		}

		if tc.wantEnableCSI {
			if len(createdDaemonset.Spec.Template.Spec.Containers) != 2 {
				t.Errorf("unexpected number of containers in daemonset:\n\t(WNT) %d\n\t(GOT): %d", len(createdDaemonset.Spec.Template.Spec.Containers), 2)
			}
		} else {
			if len(createdDaemonset.Spec.Template.Spec.Containers) != 1 {
				t.Errorf("unexpected number of containers in daemonset:\n\t(WNT) %d\n\t(GOT): %d", len(createdDaemonset.Spec.Template.Spec.Containers), 1)
			}
		}

		if tc.wantSharedDir != "" {
			sharedDirVolFound := false
			for _, vol := range createdDaemonset.Spec.Template.Spec.Volumes {
				if vol.Name == "shared" {
					sharedDirVolFound = true
					if vol.HostPath.Path != tc.wantSharedDir {
						t.Errorf("unexpected sharedDir path:\n\t(WNT) %s\n\t(GOT) %s", tc.wantSharedDir, vol.HostPath.Path)
					}
					break
				}
			}
			if !sharedDirVolFound {
				t.Errorf("expected shared volume, but not found")
			}
		}

		// Check telemetry option.
		telemetryEnvVarFound := false
		wantDisableTelemetry := strconv.FormatBool(tc.wantDisableTelemetry)
		for _, env := range createdDaemonset.Spec.Template.Spec.Containers[0].Env {
			if env.Name == disableTelemetryEnvVar {
				telemetryEnvVarFound = true
				if env.Value != wantDisableTelemetry {
					t.Errorf("unexpected disableTelemetry value:\n\t(WNT) %s\n\t(GOT) %s", wantDisableTelemetry, env.Value)
				}
			}
		}

		// Telemetry must be set.
		if !telemetryEnvVarFound {
			t.Errorf("disableTelemetry env var not set, expected to be set")
		}

		// Check fencing option.
		fencingEnvVarFound := false
		wantDisableFencing := strconv.FormatBool(tc.wantDisableFencing)
		for _, env := range createdDaemonset.Spec.Template.Spec.Containers[0].Env {
			if env.Name == disableFencingEnvVar {
				fencingEnvVarFound = true
				if env.Value != wantDisableFencing {
					t.Errorf("unexpected disableFencing value:\n\t(WNT) %s\n\t(GOT) %s", wantDisableFencing, env.Value)
				}
			}
		}

		// Fencing must be set.
		if !fencingEnvVarFound {
			t.Errorf("disableFencing env var not set, expected to be set")
		}

		// Check disable tcmu option.
		disableTCMUEnvVarFound := false
		wantDisableTCMU := strconv.FormatBool(tc.wantDisableTCMU)
		for _, env := range createdDaemonset.Spec.Template.Spec.Containers[0].Env {
			if env.Name == disableTCMUEnvVar {
				disableTCMUEnvVarFound = true
				if env.Value != wantDisableTCMU {
					t.Errorf("unexpected disableTCMU value:\n\t(WNT) %s\n\t(GOT) %s", wantDisableTCMU, env.Value)
				}
			}
		}

		// Disable TCMU must be set.
		if !disableTCMUEnvVarFound {
			t.Errorf("disableTCMU env var not set, expected to be set")
		}

		// Check force tcmu option.
		ForceTCMUEnvVarFound := false
		wantForceTCMU := strconv.FormatBool(tc.wantForceTCMU)
		for _, env := range createdDaemonset.Spec.Template.Spec.Containers[0].Env {
			if env.Name == forceTCMUEnvVar {
				ForceTCMUEnvVarFound = true
				if env.Value != wantForceTCMU {
					t.Errorf("unexpected forceTCMU value:\n\t(WNT) %s\n\t(GOT) %s", wantForceTCMU, env.Value)
				}
			}
		}

		// Force TCMU must be set.
		if !ForceTCMUEnvVarFound {
			t.Errorf("forceTCMU env var not set, expected to be set")
		}

		if tc.wantTLSEtcd {
			// Check if the TLS certs volume exists in the spec.
			volumeFound := false
			for _, vol := range createdDaemonset.Spec.Template.Spec.Volumes {
				if vol.Name == tlsEtcdCertsVolume {
					volumeFound = true
				}
			}
			if !volumeFound {
				t.Error("TLS etcd certs volume not found in daemonset spec")
			}

			// Check if TLS certs volume mount exists in the node container.
			volumeMountFound := false
			for _, volMount := range createdDaemonset.Spec.Template.Spec.Containers[0].VolumeMounts {
				if volMount.Name == tlsEtcdCertsVolume &&
					volMount.MountPath == tlsEtcdRootPath {
					volumeMountFound = true
				}
			}
			if !volumeMountFound {
				t.Error("TLS etcd certs volume mount not found in the node container")
			}

			// Check if etcd TLS certs env vars are set for the node container.
			tlsEtcdCAEnvVarFound := false
			tlsEtcdClientCertEnvVarFound := false
			tlsEtcdClientKeyEnvVarFound := false

			for _, env := range createdDaemonset.Spec.Template.Spec.Containers[0].Env {
				switch env.Name {
				case tlsEtcdCAEnvVar:
					tlsEtcdCAEnvVarFound = true
					// Check the env var value.
					wantCAPath := filepath.Join(tlsEtcdRootPath, tlsEtcdCA)
					if env.Value != wantCAPath {
						t.Errorf("unexpected %q value:\n\t(WNT) %q\n\t(GOT) %q", env.Name, wantCAPath, env.Value)
					}
				case tlsEtcdClientCertEnvVar:
					tlsEtcdClientCertEnvVarFound = true
					wantCertPath := filepath.Join(tlsEtcdRootPath, tlsEtcdClientCert)
					if env.Value != wantCertPath {
						t.Errorf("unexpected %q value:\n\t(WNT) %q\n\t(GOT) %q", env.Name, wantCertPath, env.Value)
					}
				case tlsEtcdClientKeyEnvVar:
					tlsEtcdClientKeyEnvVarFound = true
					wantKeyPath := filepath.Join(tlsEtcdRootPath, tlsEtcdClientKey)
					if env.Value != wantKeyPath {
						t.Errorf("unexpected %q value:\n\t(WNT) %q\n\t(GOT) %q", env.Name, wantKeyPath, env.Value)
					}
				}
			}

			if !tlsEtcdCAEnvVarFound {
				t.Errorf("%q env var not set, expected to be set", tlsEtcdCAEnvVar)
			}
			if !tlsEtcdClientCertEnvVarFound {
				t.Errorf("%q env var not set, expected to be set", tlsEtcdClientCertEnvVar)
			}
			if !tlsEtcdClientKeyEnvVarFound {
				t.Errorf("%q env var not set, expected to be set", tlsEtcdClientKeyEnvVar)
			}
		}

		// Check k8sDistro matches if set
		k8sDistroEnvVarFound := false
		for _, env := range createdDaemonset.Spec.Template.Spec.Containers[0].Env {
			if env.Name == k8sDistroEnvVar {
				k8sDistroEnvVarFound = true
				if env.Value != tc.wantK8sDistro {
					t.Errorf("unexpected k8sDistro value:\n\t(WNT) %s\n\t(GOT) %s", tc.wantK8sDistro, env.Value)
				}
			}
		}

		// k8sDistro must be set, but can be empty
		if !k8sDistroEnvVarFound {
			t.Errorf("k8sDistro env var not set, expected to be set")
		}

		stosCluster.Spec = api.StorageOSClusterSpec{}
		c.Delete(context.Background(), createdDaemonset)
		if err := c.Get(context.Background(), nsName, createdDaemonset); err == nil {
			t.Fatal("failed to delete the created object", err)
		}
	}
}

func TestCreateCSIHelper(t *testing.T) {
	clusterName := "my-stos-cluster"
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: defaultNS,
		},
	}

	testcases := []struct {
		name                 string
		spec                 api.StorageOSClusterSpec
		wantHelperDeployment bool // CSI helper as k8s Deployment.
	}{
		{
			name: "CSI helpers default",
			spec: api.StorageOSClusterSpec{
				CSI: api.StorageOSClusterCSI{
					Enable: true,
				},
			},
			wantHelperDeployment: false,
		},
		{
			name: "CSI helpers statefulset",
			spec: api.StorageOSClusterSpec{
				CSI: api.StorageOSClusterCSI{
					Enable:             true,
					DeploymentStrategy: statefulsetKind,
				},
			},
			wantHelperDeployment: false,
		},
		{
			name: "CSI helpers deployment",
			spec: api.StorageOSClusterSpec{
				CSI: api.StorageOSClusterCSI{
					Enable:             true,
					DeploymentStrategy: deploymentKind,
				},
			},
			wantHelperDeployment: true,
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(testScheme)
			deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)

			stosCluster.Spec = tc.spec
			if err := deploy.createCSIHelper(); err != nil {
				t.Fatal("failed to create csi helper", err)
			}

			// Get tolerations for pod toleration checks.
			var tolerations []corev1.Toleration

			if tc.wantHelperDeployment {
				// Check for Deployment resource.
				createdDeployment := &appsv1.Deployment{}
				nsNameDeployment := types.NamespacedName{
					Name:      csiHelperName,
					Namespace: defaultNS,
				}

				if err := c.Get(context.Background(), nsNameDeployment, createdDeployment); err != nil {
					t.Fatal("failed to get the created deployment", err)
				}

				tolerations = createdDeployment.Spec.Template.Spec.Tolerations
			} else {
				// Check for StatefulSet resource.
				createdStatefulset := &appsv1.StatefulSet{}
				nsNameStatefulSet := types.NamespacedName{
					Name:      statefulsetName,
					Namespace: defaultNS,
				}

				if err := c.Get(context.Background(), nsNameStatefulSet, createdStatefulset); err != nil {
					t.Fatal("failed to get the created statefulset")
				}

				tolerations = createdStatefulset.Spec.Template.Spec.Tolerations
			}

			// Check if the recovery tolerations are applied.
			foundNotReadyTol := false
			foundUnreachableTol := false
			for _, toleration := range tolerations {
				switch toleration.Key {
				case nodeNotReadyTolKey:
					foundNotReadyTol = true
				case nodeUnreachableTolKey:
					foundUnreachableTol = true
				}
			}

			if !foundNotReadyTol {
				t.Errorf("pod toleration %q not found in CSI helper", nodeNotReadyTolKey)
			}
			if !foundUnreachableTol {
				t.Errorf("pod toleration %q not found for CSI helper", nodeUnreachableTolKey)
			}
		})
	}
}

func TestDeployLegacy(t *testing.T) {
	const (
		containersCount = 1
		volumesCount    = 4
	)

	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
	}

	testCases := []struct {
		name       string
		k8sVersion string
	}{
		{
			name:       "empty",
			k8sVersion: "",
		},
		{
			name:       "1.9",
			k8sVersion: "1.9",
		},
		{
			name:       "1.11.0",
			k8sVersion: "1.11.0",
		},
		{
			name:       "1.12.2",
			k8sVersion: "1.12.2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(testScheme)
			if err := c.Create(context.Background(), stosCluster); err != nil {
				t.Fatalf("failed to create storageoscluster object: %v", err)
			}

			deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, tc.k8sVersion, false)
			if err := deploy.Deploy(); err != nil {
				t.Fatalf("failed to deploy cluster: %v", err)
			}

			createdDaemonset := &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      daemonsetName,
					Namespace: stosCluster.Spec.GetResourceNS(),
				},
			}

			nsName := types.NamespacedName{
				Name:      daemonsetName,
				Namespace: defaultNS,
			}

			if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
				t.Fatal("failed to get the created daemonset", err)
			}

			if len(createdDaemonset.Spec.Template.Spec.Containers) != containersCount {
				t.Errorf("unexpected number of containers in the DaemonSet:\n\t(GOT) %d\n\t(WNT) %d", len(createdDaemonset.Spec.Template.Spec.Containers), containersCount)
			}

			if len(createdDaemonset.Spec.Template.Spec.Volumes) != volumesCount {
				t.Errorf("unexpected number of volumes in the DaemonSet:\n\t(GOT) %d\n\t(WNT) %d", len(createdDaemonset.Spec.Template.Spec.Volumes), volumesCount)
			}
		})
	}
}

func TestDeployCSI(t *testing.T) {
	const (
		kubeletPluginsWatcherDriverRegArgsCount = 3
		containersCount                         = 2
		volumesCount                            = 9
	)

	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
		Spec: api.StorageOSClusterSpec{
			CSI: api.StorageOSClusterCSI{
				Enable: true,
			},
		},
	}

	testCases := []struct {
		name                          string
		k8sVersion                    string
		supportsKubeletPluginsWatcher bool
	}{
		{
			name:       "empty",
			k8sVersion: "",
		},
		{
			name:       "1.9.0",
			k8sVersion: "1.9.0",
		},
		{
			name:       "1.11.0",
			k8sVersion: "1.11.0",
		},
		{
			name:                          "1.12.0",
			k8sVersion:                    "1.12.0",
			supportsKubeletPluginsWatcher: true,
		},
		{
			name:                          "1.12.2",
			k8sVersion:                    "1.12.2",
			supportsKubeletPluginsWatcher: true,
		},
		{
			name:       "1.9.1+a0ce1bc657",
			k8sVersion: "1.9.1+a0ce1bc657",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c := fake.NewFakeClientWithScheme(testScheme)
			if err := c.Create(context.Background(), stosCluster); err != nil {
				t.Fatalf("failed to create storageoscluster object: %v", err)
			}

			deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, tc.k8sVersion, false)
			if err := deploy.Deploy(); err != nil {
				t.Fatalf("failed to deploy cluster: %v", err)
			}

			createdDaemonset := &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      daemonsetName,
					Namespace: stosCluster.Spec.GetResourceNS(),
				},
			}

			nsName := types.NamespacedName{
				Name:      daemonsetName,
				Namespace: defaultNS,
			}

			if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
				t.Fatal("failed to get the created daemonset", err)
			}

			if len(createdDaemonset.Spec.Template.Spec.Containers) != containersCount {
				t.Errorf("unexpected number of containers in the DaemonSet:\n\t(GOT) %d\n\t(WNT) %d", len(createdDaemonset.Spec.Template.Spec.Containers), containersCount)
			}

			if len(createdDaemonset.Spec.Template.Spec.Volumes) != volumesCount {
				t.Errorf("unexpected number of volumes in the DaemonSet:\n\t(GOT) %d\n\t(WNT) %d", len(createdDaemonset.Spec.Template.Spec.Volumes), volumesCount)
			}

			// KubeletPluginsWatcher support is only on k8s 1.12.0+.
			if kubeletPluginsWatcherSupported(tc.k8sVersion) != tc.supportsKubeletPluginsWatcher {
				t.Errorf("expected KubeletPluginsWatcherSupported to be %t", tc.supportsKubeletPluginsWatcher)
			}

			// When KubeletPluginsWatcher is supported, some extra arguments are
			// passed to set the proper registration mode.
			if kubeletPluginsWatcherSupported(tc.k8sVersion) {
				driverReg := createdDaemonset.Spec.Template.Spec.Containers[1]
				if len(driverReg.Args) != kubeletPluginsWatcherDriverRegArgsCount {
					t.Errorf("unexpected number of args for DriverRegistration container:\n\t(GOT) %d\n\t(WNT) %d", len(driverReg.Args), kubeletPluginsWatcherDriverRegArgsCount)
				}
			}
		})
	}
}

func TestDeployKVBackend(t *testing.T) {
	testKVAddr := "1.2.3.4:1111,4.3.2.1:0000"
	testBackend := "etcd"

	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
		Spec: api.StorageOSClusterSpec{
			KVBackend: api.StorageOSClusterKVBackend{
				Address: testKVAddr,
				Backend: testBackend,
			},
		},
	}

	c := fake.NewFakeClientWithScheme(testScheme)
	if err := c.Create(context.Background(), stosCluster); err != nil {
		t.Fatalf("failed to create storageoscluster object: %v", err)
	}

	deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
	if err := deploy.Deploy(); err != nil {
		t.Fatalf("failed to deploy cluster: %v", err)
	}

	createdDaemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonsetName,
			Namespace: stosCluster.Spec.GetResourceNS(),
		},
	}

	nsName := types.NamespacedName{
		Name:      daemonsetName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
		t.Fatal("failed to get the created daemonset", err)
	}

	podSpec := createdDaemonset.Spec.Template.Spec.Containers[0]

	foundKVAddr := false
	foundKVBackend := false

	for _, e := range podSpec.Env {
		switch e.Name {
		case kvAddrEnvVar:
			foundKVAddr = true
			if e.Value != testKVAddr {
				t.Errorf("unexpected %s value:\n\t(GOT) %s\n\t(WNT) %s", kvAddrEnvVar, e.Value, testKVAddr)
			}
		case kvBackendEnvVar:
			foundKVBackend = true
			if e.Value != testBackend {
				t.Errorf("unexpected %s value:\n\t(GOT) %s\n\t(WNT) %s", kvBackendEnvVar, e.Value, testBackend)
			}
		}
	}

	if !foundKVAddr {
		t.Errorf("expected %s to be in the pod spec env", kvAddrEnvVar)
	}
	if !foundKVBackend {
		t.Errorf("expected %s to be in the pod spec env", kvBackendEnvVar)
	}
}

func TestDeployDebug(t *testing.T) {
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
		Spec: api.StorageOSClusterSpec{
			Debug: true,
		},
	}

	c := fake.NewFakeClientWithScheme(testScheme)
	if err := c.Create(context.Background(), stosCluster); err != nil {
		t.Fatalf("failed to create storageoscluster object: %v", err)
	}

	deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
	if err := deploy.Deploy(); err != nil {
		t.Fatalf("failed to deploy cluster: %v", err)
	}

	createdDaemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonsetName,
			Namespace: stosCluster.Spec.GetResourceNS(),
		},
	}

	nsName := types.NamespacedName{
		Name:      daemonsetName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
		t.Fatal("failed to get the created daemonset", err)
	}

	podSpec := createdDaemonset.Spec.Template.Spec.Containers[0]

	foundDebug := false

	for _, e := range podSpec.Env {
		switch e.Name {
		case debugEnvVar:
			foundDebug = true
			if e.Value != debugVal {
				t.Errorf("unexpected %s value:\n\t(GOT) %s\n\t(WNT) %s", debugEnvVar, e.Value, debugVal)
			}
		}
	}

	if !foundDebug {
		t.Errorf("expected %s to be in the pod spec env", debugEnvVar)
	}
}

func TestDeployNodeAffinity(t *testing.T) {
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
		Spec: api.StorageOSClusterSpec{
			CSI: api.StorageOSClusterCSI{
				Enable: true,
			},
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "foo",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"baz"},
						},
					},
				},
			},
		},
	}

	testcases := []struct {
		name                  string
		csiDeploymentStrategy string
	}{
		{
			name:                  "csi helper StatefulSet",
			csiDeploymentStrategy: "statefulset",
		},
		{
			name:                  "csi helper Deployment",
			csiDeploymentStrategy: "deployment",
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			stosCluster.Spec.CSI.DeploymentStrategy = tc.csiDeploymentStrategy

			c := fake.NewFakeClientWithScheme(testScheme)
			if err := c.Create(context.Background(), stosCluster); err != nil {
				t.Fatalf("failed to create storageoscluster object: %v", err)
			}

			deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
			if err := deploy.Deploy(); err != nil {
				t.Fatalf("failed to deploy cluster: %v", err)
			}

			createdDaemonset := &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      daemonsetName,
					Namespace: stosCluster.Spec.GetResourceNS(),
				},
			}

			nsName := types.NamespacedName{
				Name:      daemonsetName,
				Namespace: defaultNS,
			}

			if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
				t.Fatal("failed to get the created daemonset", err)
			}

			podSpec := createdDaemonset.Spec.Template.Spec

			if !reflect.DeepEqual(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, stosCluster.Spec.NodeSelectorTerms) {
				t.Errorf("unexpected DaemonSet NodeSelectorTerms value:\n\t(GOT) %v\n\t(WNT) %v", stosCluster.Spec.NodeSelectorTerms, podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			}

			// Fetch and check both the CSI helpers kinds.
			if tc.csiDeploymentStrategy == "deployment" {
				createdDeployment := &appsv1.Deployment{}
				nsNameDeployment := types.NamespacedName{
					Name:      csiHelperName,
					Namespace: defaultNS,
				}

				if err := c.Get(context.Background(), nsNameDeployment, createdDeployment); err != nil {
					t.Fatal("failed to get the created CSI helper deployment", err)
				}

				podSpec = createdDeployment.Spec.Template.Spec
			} else {
				createdStatefulset := &appsv1.StatefulSet{}
				nsNameStatefulSet := types.NamespacedName{
					Name:      statefulsetName,
					Namespace: defaultNS,
				}

				if err := c.Get(context.Background(), nsNameStatefulSet, createdStatefulset); err != nil {
					t.Fatal("failed to get the created CSI helper statefulset", err)
				}

				podSpec = createdStatefulset.Spec.Template.Spec
			}

			if !reflect.DeepEqual(podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms, stosCluster.Spec.NodeSelectorTerms) {
				t.Errorf("unexpected StatefulSet NodeSelectorTerms value:\n\t(GOT) %v\n\t(WNT) %v", stosCluster.Spec.NodeSelectorTerms, podSpec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution)
			}
		})
	}
}

func TestDeployTolerations(t *testing.T) {
	testCases := []struct {
		name        string
		tolerations []corev1.Toleration
		wantError   bool
	}{
		{
			name: "TolerationOpExists without value",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: corev1.TolerationOpExists,
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
		},
		{
			name: "TolerationOpExists with value",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: corev1.TolerationOpExists,
					Value:    "bar",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
			wantError: true,
		},
		{
			name: "TolerationOpEqual",
			tolerations: []corev1.Toleration{
				{
					Key:      "foo",
					Operator: corev1.TolerationOpEqual,
					Value:    "bar",
					Effect:   corev1.TaintEffectNoSchedule,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stosCluster := &api.StorageOSCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "teststos",
					Namespace: "default",
				},
				Spec: api.StorageOSClusterSpec{
					CSI: api.StorageOSClusterCSI{
						Enable: false,
					},
					Tolerations: tc.tolerations,
				},
			}

			c := fake.NewFakeClientWithScheme(testScheme)
			if err := c.Create(context.Background(), stosCluster); err != nil {
				t.Fatalf("failed to create storageoscluster object: %v", err)
			}

			deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
			err := deploy.Deploy()
			if !tc.wantError && err != nil {
				t.Errorf("expected no error but got one: %v", err)
			}
			if tc.wantError && err == nil {
				t.Errorf("expected error but got none")
			}

			if tc.wantError {
				return
			}

			createdDaemonset := &appsv1.DaemonSet{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "apps/v1",
					Kind:       "DaemonSet",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      daemonsetName,
					Namespace: stosCluster.Spec.GetResourceNS(),
				},
			}

			nsName := types.NamespacedName{
				Name:      daemonsetName,
				Namespace: defaultNS,
			}

			if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
				t.Fatal("failed to get the created daemonset", err)
			}

			podSpec := createdDaemonset.Spec.Template.Spec

			if !reflect.DeepEqual(podSpec.Tolerations, stosCluster.Spec.Tolerations) {
				t.Errorf("unexpected Tolerations value:\n\t(GOT) %v\n\t(WNT) %v", podSpec.Tolerations, stosCluster.Spec.Tolerations)
			}
		})
	}

}

func TestDeployNodeResources(t *testing.T) {
	memLimit, _ := resource.ParseQuantity("1Gi")
	memReq, _ := resource.ParseQuantity("702Mi")
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
		Spec: api.StorageOSClusterSpec{
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceMemory: memLimit,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceMemory: memReq,
				},
			},
		},
	}

	c := fake.NewFakeClientWithScheme(testScheme)
	if err := c.Create(context.Background(), stosCluster); err != nil {
		t.Fatalf("failed to create storageoscluster object: %v", err)
	}

	deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
	if err := deploy.Deploy(); err != nil {
		t.Fatalf("failed to deploy cluster: %v", err)
	}

	createdDaemonset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      daemonsetName,
			Namespace: stosCluster.Spec.GetResourceNS(),
		},
	}

	nsName := types.NamespacedName{
		Name:      daemonsetName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
		t.Fatal("failed to get the created daemonset", err)
	}

	nodeContainer := createdDaemonset.Spec.Template.Spec.Containers[0]

	if !reflect.DeepEqual(nodeContainer.Resources.Limits, stosCluster.Spec.Resources.Limits) {
		t.Errorf("unexpected resources limits value:\n\t(GOT) %v\n\t(WNT) %v", nodeContainer.Resources.Limits, stosCluster.Spec.Resources.Limits)
	}

	if !reflect.DeepEqual(nodeContainer.Resources.Requests, stosCluster.Spec.Resources.Requests) {
		t.Errorf("unexpected resources requests value:\n\t(GOT) %v\n\t(WNT) %v", nodeContainer.Resources.Requests, stosCluster.Spec.Resources.Limits)
	}
}

func TestDelete(t *testing.T) {
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
	}

	testcases := []struct {
		name string
		spec api.StorageOSClusterSpec
	}{
		{
			name: "delete daemonset and CSI helper statefulset",
			spec: api.StorageOSClusterSpec{
				CSI: api.StorageOSClusterCSI{
					Enable:             true,
					DeploymentStrategy: "statefulset",
				},
			},
		},
		{
			name: "delete daemonset and CSI helper deployment",
			spec: api.StorageOSClusterSpec{
				CSI: api.StorageOSClusterCSI{
					Enable:             true,
					DeploymentStrategy: "deployment",
				},
			},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			stosCluster.Spec = tc.spec

			c := fake.NewFakeClientWithScheme(testScheme)
			if err := c.Create(context.Background(), stosCluster); err != nil {
				t.Fatalf("failed to create storageoscluster object: %v", err)
			}

			createdNamespace := &corev1.Namespace{}
			nsNameNamespace := types.NamespacedName{
				Name: defaultNS,
			}

			// The namespace should not exist.
			if err := c.Get(context.Background(), nsNameNamespace, createdNamespace); err == nil {
				t.Fatal("expected the namespace to not exist initially", err)
			}

			deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "1.13.0", false)
			if err := deploy.Deploy(); err != nil {
				t.Fatalf("failed to deploy cluster: %v", err)
			}

			// Check if the namespace, daemonset and statefulset have been created.
			if err := c.Get(context.Background(), nsNameNamespace, createdNamespace); err != nil {
				t.Fatal("failed to get the created namespace", err)
			}

			createdDaemonset := &appsv1.DaemonSet{}
			nsNameDaemonSet := types.NamespacedName{
				Name:      daemonsetName,
				Namespace: defaultNS,
			}

			if err := c.Get(context.Background(), nsNameDaemonSet, createdDaemonset); err != nil {
				t.Fatal("failed to get the created daemonset", err)
			}

			// Check creation and deletion of both CSI helper Deployment and
			// StatefulSet.

			var createdCSIHelperDeployment *appsv1.Deployment
			var createdCSIHelperStatefulSet *appsv1.StatefulSet

			nsNameDeployment := types.NamespacedName{
				Name:      csiHelperName,
				Namespace: defaultNS,
			}

			nsNameStatefulSet := types.NamespacedName{
				Name:      statefulsetName,
				Namespace: defaultNS,
			}

			if tc.spec.GetCSIDeploymentStrategy() == "deployment" {
				createdCSIHelperDeployment = &appsv1.Deployment{}
				if err := c.Get(context.Background(), nsNameDeployment, createdCSIHelperDeployment); err != nil {
					t.Fatal("failed to get the created statefulset", err)
				}
			} else {
				createdCSIHelperStatefulSet = &appsv1.StatefulSet{}
				if err := c.Get(context.Background(), nsNameStatefulSet, createdCSIHelperStatefulSet); err != nil {
					t.Fatal("failed to get the created statefulset", err)
				}
			}

			// Delete the deployment.
			if err := deploy.Delete(); err != nil {
				t.Fatalf("failed to delete cluster: %v", err)
			}

			// Daemonset and statefulset should have been deleted.
			if err := c.Get(context.Background(), nsNameDaemonSet, createdDaemonset); err == nil {
				t.Fatal("expected the daemonset to be deleted, but it still exists")
			}

			// Check CSI helper deletion.
			if tc.spec.GetCSIDeploymentStrategy() == "deployment" {
				if err := c.Get(context.Background(), nsNameDeployment, createdCSIHelperDeployment); err == nil {
					t.Fatal("expected the CSI helper deployment to be deleted, but it still exists")
				}
			} else {
				if err := c.Get(context.Background(), nsNameStatefulSet, createdCSIHelperStatefulSet); err == nil {
					t.Fatal("expected the CSI helper statefulset to be deleted, but it still exists")
				}
			}

			// The namespace should not be deleted.
			if err := c.Get(context.Background(), nsNameNamespace, createdNamespace); err != nil {
				t.Fatal("failed to get the created namespace", err)
			}
		})
	}
}

// TestDeployTLSEtcdCerts deploys a storageos cluster with etcd TLS certs secret
// reference, checks if a new secret is created in the namespace where
// storageos resources are created and verifies that the secret has the same
// data as the source secret.
func TestDeployTLSEtcdCerts(t *testing.T) {
	// etcd secret containing TLS certs. This exists before storageos cluster
	// is created.
	etcdSecret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "etcd-certs",
			Namespace: "default",
		},
		Data: map[string][]byte{
			tlsEtcdCA:         []byte("someetcdca"),
			tlsEtcdClientCert: []byte("someetcdclientcert"),
			tlsEtcdClientKey:  []byte("someetcdclientkey"),
		},
	}

	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
		Spec: api.StorageOSClusterSpec{
			TLSEtcdSecretRefName:      "etcd-certs",
			TLSEtcdSecretRefNamespace: "default",
			CSI: api.StorageOSClusterCSI{
				Enable: true,
			},
		},
	}

	// Create fake client with existing etcd TLS secret.
	c := fake.NewFakeClientWithScheme(testScheme, etcdSecret)
	if err := c.Create(context.Background(), stosCluster); err != nil {
		t.Fatalf("failed to create storageoscluster object: %v", err)
	}

	// Deploy storageos cluster.
	deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
	if err := deploy.Deploy(); err != nil {
		t.Fatalf("failed to deploy cluster: %v", err)
	}

	// Get the secret created by the deployment.
	stosEtcdSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TLSEtcdSecretName,
			Namespace: "storageos",
		},
	}
	nsName := types.NamespacedName{
		Name:      TLSEtcdSecretName,
		Namespace: "storageos",
	}
	if err := c.Get(context.Background(), nsName, stosEtcdSecret); err != nil {
		t.Fatalf("expected %q secret to exist, but not found", stosEtcdSecret)
	}

	// Check the created secret type because the fake k8s client doesn't
	// validate the type of secret and the data fields.
	// For example, it allows creating a TLS type secret with opaque type data.
	// TLS type secret can have only `tls.key` and `tls.crt` data fields.
	if etcdSecret.Type != stosEtcdSecret.Type {
		t.Errorf("unexpected secret type:\n\t(WNT) %s\n\t(GOT) %s", etcdSecret.Type, stosEtcdSecret.Type)
	}

	// Check if the data in the new secret is the same as the source secret.
	if !reflect.DeepEqual(stosEtcdSecret.Data, etcdSecret.Data) {
		t.Errorf("unexpected secret data:\n\t(WNT) %v\n\t(GOT) %v", etcdSecret.Data, stosEtcdSecret.Data)
	}
}

// TestDeployPodPriorityClass tests that the pod priority class is set properly
// for the daemonset and statefulset pods when deployed in kube-system
// namespace.
func TestDeployPodPriorityClass(t *testing.T) {
	testCases := []struct {
		name                  string
		resourceNS            string
		csiDeploymentStrategy string
		wantPriorityClass     bool
	}{
		{
			name:                  "have priority class set | CSI StatefulSet",
			resourceNS:            "kube-system",
			csiDeploymentStrategy: "statefulset",
			wantPriorityClass:     true,
		},
		{
			name:                  "have priority class set | CSI Deployment",
			resourceNS:            "kube-system",
			csiDeploymentStrategy: "deployment",
			wantPriorityClass:     true,
		},
		{
			name:              "no priority class set",
			resourceNS:        "storageos",
			wantPriorityClass: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stosCluster := &api.StorageOSCluster{
				TypeMeta: metav1.TypeMeta{
					APIVersion: gvk.GroupVersion().String(),
					Kind:       gvk.Kind,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "teststos",
					Namespace: "default",
				},
				Spec: api.StorageOSClusterSpec{
					CSI: api.StorageOSClusterCSI{
						Enable:             true,
						DeploymentStrategy: tc.csiDeploymentStrategy,
					},
					Namespace: tc.resourceNS,
				},
			}

			c := fake.NewFakeClientWithScheme(testScheme)
			if err := c.Create(context.Background(), stosCluster); err != nil {
				t.Fatalf("failed to create storageoscluster object: %v", err)
			}

			deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "", false)
			if err := deploy.Deploy(); err != nil {
				t.Fatalf("failed to deploy cluster: %v", err)
			}

			// Check daemonset pod priority class.
			createdDaemonset := &appsv1.DaemonSet{}

			nsName := types.NamespacedName{
				Name:      daemonsetName,
				Namespace: stosCluster.Spec.GetResourceNS(),
			}

			if err := c.Get(context.Background(), nsName, createdDaemonset); err != nil {
				t.Fatal("failed to get the created daemonset", err)
			}

			daemonsetPC := createdDaemonset.Spec.Template.Spec.PriorityClassName
			if tc.wantPriorityClass && daemonsetPC != criticalPriorityClass {
				t.Errorf("unexpected daemonset pod priodity class:\n\t(GOT) %v \n\t(WNT) %v", daemonsetPC, criticalPriorityClass)
			}

			if !tc.wantPriorityClass && daemonsetPC != "" {
				t.Errorf("expected daemonset priority class to be not set")
			}

			// Check pod priority class for both the kinds of CSI helpers.
			var csiHelperPC string

			if stosCluster.Spec.GetCSIDeploymentStrategy() == "deployment" {
				createdDeployment := &appsv1.Deployment{}
				nsNameDeployment := types.NamespacedName{
					Name:      csiHelperName,
					Namespace: stosCluster.Spec.GetResourceNS(),
				}

				if err := c.Get(context.Background(), nsNameDeployment, createdDeployment); err != nil {
					t.Fatal("failed to get the created CSI helper deployment", err)
				}

				csiHelperPC = createdDeployment.Spec.Template.Spec.PriorityClassName
			} else {
				createdStatefulset := &appsv1.StatefulSet{}
				nsNameStatefulSet := types.NamespacedName{
					Name:      statefulsetName,
					Namespace: stosCluster.Spec.GetResourceNS(),
				}

				if err := c.Get(context.Background(), nsNameStatefulSet, createdStatefulset); err != nil {
					t.Fatal("failed to get the created CSI helper statefulset", err)
				}

				csiHelperPC = createdStatefulset.Spec.Template.Spec.PriorityClassName
			}

			if tc.wantPriorityClass && csiHelperPC != criticalPriorityClass {
				t.Errorf("unexpected CSI helper pod priodity class:\n\t(GOT) %v \n\t(WNT) %v", daemonsetPC, criticalPriorityClass)
			}

			if !tc.wantPriorityClass && csiHelperPC != "" {
				t.Errorf("expected CSI helper priority class to be not set")
			}
		})
	}

}

func TestDeploySchedulerExtender(t *testing.T) {
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "teststos",
			Namespace: "default",
		},
		Spec: api.StorageOSClusterSpec{
			CSI: api.StorageOSClusterCSI{
				Enable: true,
			},
		},
	}

	c := fake.NewFakeClientWithScheme(testScheme)
	if err := c.Create(context.Background(), stosCluster); err != nil {
		t.Fatalf("failed to create storageoscluster object: %v", err)
	}

	deploy := NewDeployment(c, stosCluster, nil, nil, testScheme, "1.15.0", false)
	err := deploy.Deploy()
	if err != nil {
		t.Error("deployment failed:", err)
	}

	// Get scheduler policy configmap and check the data.
	policycm := &corev1.ConfigMap{}
	policyNSName := types.NamespacedName{
		Name:      policyConfigMapName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), policyNSName, policycm); err != nil {
		t.Fatal("failed to get the created scheduler policy configmap", err)
	}

	// Check if the expected key and value exists.
	if val, exists := policycm.Data[policyConfigKey]; exists {
		if len(val) == 0 {
			t.Errorf("%q is empty, expected not to be empty", policyConfigKey)
		}
	} else {
		t.Errorf("expected %q to be in scheduler policy configmap data", policyConfigKey)
	}

	// Get scheduler configuration configmap and check the data.
	schedConfigcm := &corev1.ConfigMap{}
	schedConfigNSName := types.NamespacedName{
		Name:      schedulerConfigConfigMapName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), schedConfigNSName, schedConfigcm); err != nil {
		t.Fatal("failed to get the created scheduler configuration configmap", err)
	}

	// Check if the expected key and value exists.
	if val, exists := schedConfigcm.Data[schedulerConfigKey]; exists {
		if len(val) == 0 {
			t.Errorf("%q is empty, expected not to be empty", schedulerConfigKey)
		}
	} else {
		t.Errorf("expected %q to be in scheduler configuration configmap data", schedulerConfigKey)
	}

	// Check the attributes of the scheduler deployment.
	schedDeployment := &appsv1.Deployment{}
	schedDeploymentNSName := types.NamespacedName{
		Name:      SchedulerExtenderName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), schedDeploymentNSName, schedDeployment); err != nil {
		t.Fatal("failed to get the created scheduler deployment", err)
	}

	if *schedDeployment.Spec.Replicas != schedulerReplicas {
		t.Fatalf("unexpected number of replicas:\n\t(WNT) %d\n\t(GOT) %d", *schedDeployment.Spec.Replicas, schedulerReplicas)
	}

	if schedDeployment.Spec.Template.Spec.ServiceAccountName != SchedulerSA {
		t.Fatalf("unexpected service account name:\n\t(WNT) %q\n\t(GOT) %q", schedDeployment.Spec.Template.Spec.ServiceAccountName, SchedulerSA)
	}
}

func TestGetNodeIPs(t *testing.T) {
	tests := []struct {
		name  string
		nodes []corev1.Node
		want  []string
	}{
		{
			name: "single node single internal ip",
			nodes: []corev1.Node{
				corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{
							corev1.NodeAddress{
								Type:    corev1.NodeInternalIP,
								Address: "1.1.1.1",
							},
						},
					},
				},
			},
			want: []string{"1.1.1.1"},
		},
		{
			name: "multiple node single internal ip",
			nodes: []corev1.Node{
				corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{
							corev1.NodeAddress{
								Type:    corev1.NodeInternalIP,
								Address: "1.1.1.1",
							},
						},
					},
				},
				corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{
							corev1.NodeAddress{
								Type:    corev1.NodeInternalIP,
								Address: "2.2.2.2",
							},
						},
					},
				},
			},
			want: []string{"1.1.1.1", "2.2.2.2"},
		},
		{
			name: "single node no address",
			nodes: []corev1.Node{
				corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{},
					},
				},
			},
			want: []string{},
		},
		{
			name: "single node multiple addresses",
			nodes: []corev1.Node{
				corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{
							corev1.NodeAddress{
								Type:    corev1.NodeHostName,
								Address: "hostA",
							},
							corev1.NodeAddress{
								Type:    corev1.NodeInternalIP,
								Address: "1.1.1.1",
							},
							corev1.NodeAddress{
								Type:    corev1.NodeExternalIP,
								Address: "2.2.2.2",
							},
						},
					},
				},
			},
			want: []string{"1.1.1.1"},
		},
		{
			name: "single node no internal ip",
			nodes: []corev1.Node{
				corev1.Node{
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{
							corev1.NodeAddress{
								Type:    corev1.NodeHostName,
								Address: "hostA",
							},
							corev1.NodeAddress{
								Type:    corev1.NodeExternalIP,
								Address: "2.2.2.2",
							},
						},
					},
				},
			},
			want: []string{"hostA"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetNodeIPs(tt.nodes); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetNodeIPs() = %v, want %v", got, tt.want)
			}
		})
	}
}
