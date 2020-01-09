package storageos

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
	fakediscovery "k8s.io/client-go/discovery/fake"
	fakeclientset "k8s.io/client-go/kubernetes/fake"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/storageos/cluster-operator/internal/pkg/image"
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

// getFakeDiscoveryClient returns a discovery client with pre-defined resource
// list.
func getFakeDiscoveryClient() (discovery.DiscoveryInterface, error) {
	client := fakeclientset.NewSimpleClientset()
	fakeDiscovery, ok := client.Discovery().(*fakediscovery.FakeDiscovery)
	if !ok {
		return nil, errors.New("could not covert Discovery() to *FakeDiscovery")
	}
	fakeDiscovery.Resources = []*metav1.APIResourceList{
		{
			// CSIDriver for CSI deployment built-in resource discovery.
			GroupVersion: "storage.k8s.io/v1beta1",
			APIResources: []metav1.APIResource{
				{Kind: "CSIDriver"},
			},
		},
	}
	return client.Discovery(), nil
}

func setupFakeDeployment() (client.Client, *Deployment, error) {
	c := fake.NewFakeClientWithScheme(testScheme)
	deploy, err := setupFakeDeploymentWithClient(c)
	return c, deploy, err
}

func setupFakeDeploymentWithClient(c client.Client) (*Deployment, error) {
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
	}
	return setupFakeDeploymentWithClientAndCluster(c, stosCluster)
}

func setupFakeDeploymentWithClientAndCluster(c client.Client, stosCluster *api.StorageOSCluster) (*Deployment, error) {
	dc, err := getFakeDiscoveryClient()
	if err != nil {
		return nil, err
	}
	deploy := NewDeployment(c, dc, stosCluster, nil, nil, testScheme, "", false)
	return deploy, nil
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
	c, deploy, err := setupFakeDeployment()
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}
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
		deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
		if err != nil {
			t.Fatalf("failed to create deployment: %v", err)
		}
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
			deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
			if err != nil {
				t.Fatalf("failed to create deployment: %v", err)
			}

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
		volumesCount    = 5 // includes ConfigMap volume
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

			dc, err := getFakeDiscoveryClient()
			if err != nil {
				t.Fatalf("failed to create discovery client: %v", err)
			}

			deploy := NewDeployment(c, dc, stosCluster, nil, nil, testScheme, tc.k8sVersion, false)
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
		volumesCount                            = 10 //Includes ConfigMap volume
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

			dc, err := getFakeDiscoveryClient()
			if err != nil {
				t.Fatalf("failed to create discovery client: %v", err)
			}

			deploy := NewDeployment(c, dc, stosCluster, nil, nil, testScheme, tc.k8sVersion, false)
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

	deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}
	if err := deploy.Deploy(); err != nil {
		t.Fatalf("failed to deploy cluster: %v", err)
	}

	createdConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: stosCluster.Spec.GetResourceNS(),
		},
	}

	cmNamespacedName := types.NamespacedName{
		Name:      configmapName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), cmNamespacedName, createdConfigMap); err != nil {
		t.Fatal("failed to get the created configmap", err)
	}

	foundKVAddr := false
	foundKVBackend := false

	for k, v := range createdConfigMap.Data {
		switch k {
		case kvAddrEnvVar:
			foundKVAddr = true
			if v != testKVAddr {
				t.Errorf("unexpected %s value:\n\t(GOT) %s\n\t(WNT) %s", etcdEndpointsEnvVar, v, testKVAddr)
			}
		case kvBackendEnvVar:
			foundKVBackend = true
			if v != testBackend {
				t.Errorf("unexpected %s value:\n\t(GOT) %s\n\t(WNT) %s", kvBackendEnvVar, v, testBackend)
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

	deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}
	if err := deploy.Deploy(); err != nil {
		t.Fatalf("failed to deploy cluster: %v", err)
	}

	createdConfigMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "core/v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configmapName,
			Namespace: stosCluster.Spec.GetResourceNS(),
		},
	}

	cmNamespacedName := types.NamespacedName{
		Name:      configmapName,
		Namespace: defaultNS,
	}

	if err := c.Get(context.Background(), cmNamespacedName, createdConfigMap); err != nil {
		t.Fatal("failed to get the created configmap", err)
	}

	foundDebug := false

	for k, v := range createdConfigMap.Data {
		switch k {
		case logLevelEnvVar:
			foundDebug = true
			if v != debugVal {
				t.Errorf("unexpected %s value:\n\t(GOT) %s\n\t(WNT) %s", logLevelEnvVar, v, debugVal)
			}
		}
	}

	if !foundDebug {
		t.Errorf("expected %s to be in the pod spec env", logLevelEnvVar)
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

			deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
			if err != nil {
				t.Fatalf("failed to create deployment: %v", err)
			}
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

			deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
			if err != nil {
				t.Fatalf("failed to create deployment: %v", err)
			}
			err = deploy.Deploy()
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

	deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}
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

			deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
			if err != nil {
				t.Fatalf("failed to create deployment: %v", err)
			}
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
	deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
	if err != nil {
		t.Fatalf("failed to create deployment: %v", err)
	}
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

			deploy, err := setupFakeDeploymentWithClientAndCluster(c, stosCluster)
			if err != nil {
				t.Fatalf("failed to create deployment: %v", err)
			}
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

	dc, err := getFakeDiscoveryClient()
	if err != nil {
		t.Fatalf("failed to create discovery client: %v", err)
	}

	deploy := NewDeployment(c, dc, stosCluster, nil, nil, testScheme, "1.15.0", false)
	err = deploy.Deploy()
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

// TestContainerImageSelection checks which container images are selected for
// StorageOS deployment based on env vars and StorageOSCluster config.
// This test should be moved to a separate package with the Get Image functions
// api package.
func TestContainerImageSelection(t *testing.T) {
	// Constants to be used to associate image with their get image function.
	const (
		storageOSNodeImage             = "StorageOSNode"
		storageOSInitImage             = "StorageOSInit"
		csiNodeDriverRegistrarImage    = "CSINodeDriverRegistrar"
		csiClusterDriverRegistrarImage = "CSIClusterDriverRegistrar"
		csiExternalProvisionerImage    = "CSIExternalProvisioner"
		csiExternalAttacherImage       = "CSIExternalAttacher"
		csiLivenessProbeImage          = "CSILivenessProbe"
		kubeSchedulerImage             = "KubeScheduler"
		nfsImage                       = "NFS"
	)

	// Given image name, cluster spec and k8s version, return the appropriate
	// image.
	getImage := func(name string, spec api.StorageOSClusterSpec, k8sVersion string) string {
		csiV1Supported := CSIV1Supported(k8sVersion)
		attacherV2Supported := CSIExternalAttacherV2Supported(k8sVersion)

		switch name {
		case storageOSNodeImage:
			return spec.GetNodeContainerImage()
		case storageOSInitImage:
			return spec.GetInitContainerImage()
		case csiClusterDriverRegistrarImage:
			return spec.GetCSIClusterDriverRegistrarImage()
		case csiNodeDriverRegistrarImage:
			return spec.GetCSINodeDriverRegistrarImage(csiV1Supported)
		case csiExternalProvisionerImage:
			return spec.GetCSIExternalProvisionerImage(csiV1Supported)
		case csiExternalAttacherImage:
			return spec.GetCSIExternalAttacherImage(csiV1Supported, attacherV2Supported)
		case csiLivenessProbeImage:
			return spec.GetCSILivenessProbeImage()
		case kubeSchedulerImage:
			return spec.GetKubeSchedulerImage(k8sVersion)
		case nfsImage:
			return spec.GetNFSServerImage()
		default:
			return ""
		}
	}

	testcases := []struct {
		name        string
		envVars     map[string]string
		clusterSpec api.StorageOSClusterSpec
		k8sVersion  string
		wantImages  map[string]string
	}{
		{
			name: "images from env var - k8s 1.13",
			envVars: map[string]string{
				image.StorageOSNodeImageEnvVar:               "foo/node:1",
				image.StorageOSInitImageEnvVar:               "foo/init:1",
				image.CSIv1ClusterDriverRegistrarImageEnvVar: "foo/cdr:1",
				image.CSIv1NodeDriverRegistrarImageEnvVar:    "foo/ndr:1",
				image.CSIv1ExternalProvisionerImageEnvVar:    "foo/ep:1",
				// k8s 1.13 supports CSI external attacher v1 only.
				image.CSIv1ExternalAttacherImageEnvVar: "foo/ea:1",
				image.CSIv1LivenessProbeImageEnvVar:    "foo/lp:1",
				image.KubeSchedulerImageEnvVar:         "foo/ks:1",
				image.NFSImageEnvVar:                   "foo/nfs:1",
			},
			k8sVersion: "1.13.0",
			wantImages: map[string]string{
				storageOSNodeImage:             "foo/node:1",
				storageOSInitImage:             "foo/init:1",
				csiClusterDriverRegistrarImage: "foo/cdr:1",
				csiNodeDriverRegistrarImage:    "foo/ndr:1",
				csiExternalProvisionerImage:    "foo/ep:1",
				csiExternalAttacherImage:       "foo/ea:1",
				csiLivenessProbeImage:          "foo/lp:1",
				kubeSchedulerImage:             "foo/ks:1",
				nfsImage:                       "foo/nfs:1",
			},
		},
		{
			name: "images override from cluster spec - k8s 1.13",
			envVars: map[string]string{
				image.StorageOSNodeImageEnvVar:               "foo/node:1",
				image.StorageOSInitImageEnvVar:               "foo/init:1",
				image.CSIv1ClusterDriverRegistrarImageEnvVar: "foo/cdr:1",
				image.CSIv1NodeDriverRegistrarImageEnvVar:    "foo/ndr:1",
				image.CSIv1ExternalProvisionerImageEnvVar:    "foo/ep:1",
				// k8s 1.13 supports CSI external attacher v1 only.
				image.CSIv1ExternalAttacherImageEnvVar: "foo/ea:1",
				image.CSIv1LivenessProbeImageEnvVar:    "foo/lp:1",
				image.KubeSchedulerImageEnvVar:         "foo/ks:1",
				image.NFSImageEnvVar:                   "foo/nfs:1",
			},
			clusterSpec: api.StorageOSClusterSpec{
				Images: api.ContainerImages{
					NodeContainer:                      "zoo/node:1",
					InitContainer:                      "zoo/init:1",
					CSIClusterDriverRegistrarContainer: "zoo/cdr:1",
					CSINodeDriverRegistrarContainer:    "zoo/ndr:1",
					CSIExternalProvisionerContainer:    "zoo/ep:1",
					CSIExternalAttacherContainer:       "zoo/ea:1",
					CSILivenessProbeContainer:          "zoo/lp:1",
					KubeSchedulerContainer:             "zoo/ks:1",
					NFSContainer:                       "zoo/nfs:1",
				},
			},
			k8sVersion: "1.13.0",
			wantImages: map[string]string{
				storageOSNodeImage:             "zoo/node:1",
				storageOSInitImage:             "zoo/init:1",
				csiClusterDriverRegistrarImage: "zoo/cdr:1",
				csiNodeDriverRegistrarImage:    "zoo/ndr:1",
				csiExternalProvisionerImage:    "zoo/ep:1",
				csiExternalAttacherImage:       "zoo/ea:1",
				csiLivenessProbeImage:          "zoo/lp:1",
				kubeSchedulerImage:             "zoo/ks:1",
				nfsImage:                       "zoo/nfs:1",
			},
		},
		{
			name:       "no env vars, no overrides, fallback images - k8s 1.13",
			k8sVersion: "1.13.0",
			wantImages: map[string]string{
				storageOSNodeImage:             image.DefaultNodeContainerImage,
				storageOSInitImage:             image.DefaultInitContainerImage,
				csiClusterDriverRegistrarImage: image.CSIv1ClusterDriverRegistrarContainerImage,
				csiNodeDriverRegistrarImage:    image.CSIv1NodeDriverRegistrarContainerImage,
				csiExternalProvisionerImage:    image.CSIv1ExternalProvisionerContainerImage,
				csiExternalAttacherImage:       image.CSIv1ExternalAttacherContainerImage,
				csiLivenessProbeImage:          image.CSIv1LivenessProbeContainerImage,
				kubeSchedulerImage:             fmt.Sprintf("%s:%s", image.DefaultKubeSchedulerContainerRegistry, "v1.13.0"),
				nfsImage:                       image.DefaultNFSContainerImage,
			},
		},
		{
			name: "env var images - k8s 1.12 - CSIv0",
			envVars: map[string]string{
				image.CSIv0DriverRegistrarImageEnvVar:     "foo/dr:1",
				image.CSIv0ExternalProvisionerImageEnvVar: "foo/ep:1",
				image.CSIv0ExternalAttacherImageEnvVar:    "foo/ea:1",
			},
			k8sVersion: "1.12.0",
			// Only relevant images.
			// Use CSI v0 helper images.
			wantImages: map[string]string{
				csiNodeDriverRegistrarImage: "foo/dr:1",
				csiExternalProvisionerImage: "foo/ep:1",
				csiExternalAttacherImage:    "foo/ea:1",
			},
		},
		{
			name: "env var images - k8s 1.14 - CSIv1",
			envVars: map[string]string{
				image.CSIv1ExternalAttacherv2ImageEnvVar: "foo/ea:2",
			},
			k8sVersion: "1.14.0",
			// Only relevant images.
			// Use attacher v2 image.
			wantImages: map[string]string{
				csiExternalAttacherImage: "foo/ea:2",
			},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			for k, v := range tc.envVars {
				os.Setenv(k, v)
			}

			defer func() {
				for k := range tc.envVars {
					os.Unsetenv(k)
				}
			}()

			for imgName, wantImg := range tc.wantImages {
				gotImg := getImage(imgName, tc.clusterSpec, tc.k8sVersion)
				if gotImg != wantImg {
					t.Errorf("unexpected image selected for %s:\n\t(WNT) %s\n\t(GOT) %s", imgName, wantImg, gotImg)
				}
			}
		})
	}
}
