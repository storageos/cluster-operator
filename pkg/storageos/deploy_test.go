package storageos

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/storageos/cluster-operator/pkg/apis/cluster/v1alpha1"
)

var gvk = schema.GroupVersionKind{
	Group:   "storageos.com",
	Version: "v1alpha1",
	Kind:    "StorageOSCluster",
}

const defaultNS = "storageos"

func setupFakeDeployment() (client.Client, *Deployment) {
	c := fake.NewFakeClient()
	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gvk.GroupVersion().String(),
			Kind:       gvk.Kind,
		},
	}

	deploy := NewDeployment(c, stosCluster, nil, "")
	return c, deploy
}

func checkObjectOwner(t *testing.T, owner metav1.OwnerReference, wantGVK schema.GroupVersionKind) {
	if owner.APIVersion != wantGVK.GroupVersion().String() {
		t.Errorf("unexpected object owner api version:\n\t(WNT) %s\n\t(GOT) %s", wantGVK.Version, owner.APIVersion)
	}
	if owner.Kind != wantGVK.Kind {
		t.Errorf("unexpected object owner kindL\n\t(WNT) %s\n\t(GOT) %s", wantGVK.Kind, owner.Kind)
	}
}

func TestCreateNamespace(t *testing.T) {
	c, deploy := setupFakeDeployment()
	if err := deploy.createNamespace(); err != nil {
		t.Fatal("failed to create namespace", err)
	}

	// Fetch the created namespace and check if it's a child of StorageOSCluster.
	nsName := types.NamespacedName{Name: defaultNS}
	wantNS := &v1.Namespace{}
	if err := c.Get(context.TODO(), nsName, wantNS); err != nil {
		t.Fatal("failed to get the created object", err)
	}

	owner := wantNS.GetOwnerReferences()[0]
	checkObjectOwner(t, owner, gvk)
}

func TestCreateServiceAccount(t *testing.T) {
	c, deploy := setupFakeDeployment()
	saName := "my-service-account"
	if err := deploy.createServiceAccount(saName); err != nil {
		t.Fatal("failed to create service account for daemonset", err)
	}

	nsName := types.NamespacedName{
		Name:      saName,
		Namespace: defaultNS,
	}
	wantServiceAccount := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: defaultNS,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
	if err := c.Get(context.TODO(), nsName, wantServiceAccount); err != nil {
		t.Fatal("failed to get the created object", err)
	}

	owner := wantServiceAccount.GetOwnerReferences()[0]
	checkObjectOwner(t, owner, gvk)
}

func TestCreateRoleForKeyMgmt(t *testing.T) {
	c, deploy := setupFakeDeployment()
	if err := deploy.createRoleForKeyMgmt(); err != nil {
		t.Fatal("failed to create role binding for key management", err)
	}

	nsName := types.NamespacedName{
		Name:      "key-management-role",
		Namespace: defaultNS,
	}
	wantRole := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "key-management-role",
			Namespace: defaultNS,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
	if err := c.Get(context.TODO(), nsName, wantRole); err != nil {
		t.Fatal("failed to get the created object", err)
	}

	owner := wantRole.GetOwnerReferences()[0]
	checkObjectOwner(t, owner, gvk)
}

func TestCreateClusterRole(t *testing.T) {
	c, deploy := setupFakeDeployment()
	roleName := "my-cluster-role"
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{""},
			Resources: []string{"nodes"},
			Verbs:     []string{"get", "update"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"events"},
			Verbs:     []string{"list", "watch", "create", "update", "patch"},
		},
	}
	if err := deploy.createClusterRole(roleName, rules); err != nil {
		t.Fatal("failed to create cluster role", err)
	}

	nsName := types.NamespacedName{
		Name: roleName,
	}
	createdClusterRole := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: roleName,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
	if err := c.Get(context.TODO(), nsName, createdClusterRole); err != nil {
		t.Fatal("failed to get the created object", err)
	}

	owner := createdClusterRole.GetOwnerReferences()[0]
	checkObjectOwner(t, owner, gvk)
	checkRulesEquality(t, rules, createdClusterRole.Rules)
}

func checkRulesEquality(t *testing.T, wantRules, gotRules []rbacv1.PolicyRule) {
	for index, wantRule := range wantRules {
		gotRule := gotRules[index]
		if !reflect.DeepEqual(wantRule.APIGroups, gotRule.APIGroups) ||
			!reflect.DeepEqual(wantRule.Resources, gotRule.Resources) ||
			!reflect.DeepEqual(wantRule.Verbs, gotRule.Verbs) {
			t.Errorf("unequal rules:\n\t(WNT) %v\n\t(GOT) %v", wantRules, gotRules)
		}
	}
}

func TestCreateRoleBindingForKeyMgmt(t *testing.T) {
	c, deploy := setupFakeDeployment()
	if err := deploy.createRoleBindingForKeyMgmt(); err != nil {
		t.Fatal("failed to create role binding", err)
	}

	nsName := types.NamespacedName{
		Name:      "key-management-binding",
		Namespace: defaultNS,
	}
	createdRoleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "key-management-binding",
			Namespace: defaultNS,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
	if err := c.Get(context.Background(), nsName, createdRoleBinding); err != nil {
		t.Fatal("failed to get the created role binding", err)
	}

	owner := createdRoleBinding.GetOwnerReferences()[0]
	checkObjectOwner(t, owner, gvk)
}

func TestCreateClusterRoleBinding(t *testing.T) {
	c, deploy := setupFakeDeployment()
	bindingName := "my-cluster-binding"
	subjects := []rbacv1.Subject{
		{
			Kind:      "ServiceAccount",
			Name:      "storageos-daemonset-sa",
			Namespace: defaultNS,
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     "driver-registrar-role",
		APIGroup: "rbac.authorization.k8s.io",
	}
	if err := deploy.createClusterRoleBinding(bindingName, subjects, roleRef); err != nil {
		t.Fatal("failed to create cluster role binding", err)
	}

	nsName := types.NamespacedName{
		Name: bindingName,
	}
	createdClusterRoleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: bindingName,
			Labels: map[string]string{
				"app": appName,
			},
		},
	}
	if err := c.Get(context.Background(), nsName, createdClusterRoleBinding); err != nil {
		t.Fatal("failed to get the created object", err)
	}

	owner := createdClusterRoleBinding.GetOwnerReferences()[0]
	checkObjectOwner(t, owner, gvk)
	checkSubjectsEquality(t, subjects, createdClusterRoleBinding.Subjects)

	if (createdClusterRoleBinding.RoleRef.Kind != roleRef.Kind) ||
		(createdClusterRoleBinding.RoleRef.Name != roleRef.Name) ||
		(createdClusterRoleBinding.RoleRef.APIGroup != roleRef.APIGroup) {
		t.Errorf("unequal role ref:\n\t(WNT) %v\n\t(GOT) %v", roleRef, createdClusterRoleBinding)
	}
}

func checkSubjectsEquality(t *testing.T, wantSubjects, gotSubjects []rbacv1.Subject) {
	for index, wantSubject := range wantSubjects {
		gotSubject := gotSubjects[index]
		if !reflect.DeepEqual(wantSubject.Kind, gotSubject.Kind) ||
			!reflect.DeepEqual(wantSubject.Name, gotSubject.Name) ||
			!reflect.DeepEqual(wantSubject.Namespace, gotSubject.Namespace) {
			t.Errorf("unequal subjects:\n\t(WNT) %v\n\t(GOT) %v", wantSubjects, gotSubjects)
		}
	}
}

func TestCreateDaemonSet(t *testing.T) {
	c := fake.NewFakeClient()
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
		name      string
		spec      api.StorageOSSpec
		enableCSI bool
		sharedDir string
	}{
		{
			name: "legacy-daemonset",
			spec: api.StorageOSSpec{
				SecretRefName:      "foo-secret",
				SecretRefNamespace: "default",
			},
		},
		{
			name: "csi-daemonset",
			spec: api.StorageOSSpec{
				SecretRefName:      "foo-secret",
				SecretRefNamespace: "default",
				CSI: api.StorageOSCSI{
					Enable: true,
				},
			},
			enableCSI: true,
		},
		{
			name: "shared-dir",
			spec: api.StorageOSSpec{
				SharedDir: "some-dir-path",
			},
			sharedDir: "some-dir-path",
		},
	}

	for _, tc := range testcases {
		stosCluster.Spec = tc.spec
		deploy := NewDeployment(c, stosCluster, nil, "")
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

		owner := createdDaemonset.GetOwnerReferences()[0]
		checkObjectOwner(t, owner, gvk)

		if tc.enableCSI {
			if len(createdDaemonset.Spec.Template.Spec.Containers) != 2 {
				t.Errorf("unexpected number of containers in daemonset:\n\t(WNT) %d\n\t(GOT): %d", len(createdDaemonset.Spec.Template.Spec.Containers), 2)
			}
		} else {
			if len(createdDaemonset.Spec.Template.Spec.Containers) != 1 {
				t.Errorf("unexpected number of containers in daemonset:\n\t(WNT) %d\n\t(GOT): %d", len(createdDaemonset.Spec.Template.Spec.Containers), 1)
			}
		}

		if tc.sharedDir != "" {
			sharedDirVolFound := false
			for _, vol := range createdDaemonset.Spec.Template.Spec.Volumes {
				if vol.Name == "shared" {
					sharedDirVolFound = true
					if vol.HostPath.Path != tc.sharedDir {
						t.Errorf("unexpected sharedDir path:\n\t(WNT) %s\n\t(GOT) %s", tc.sharedDir, vol.HostPath.Path)
					}
					break
				}
			}
			if !sharedDirVolFound {
				t.Errorf("expected shared volume, but not found")
			}
		}

		stosCluster.Spec = api.StorageOSSpec{}
		c.Delete(context.Background(), createdDaemonset)
		if err := c.Get(context.Background(), nsName, createdDaemonset); err == nil {
			t.Fatal("failed to delete the created object", err)
		}
	}
}

func TestCreateStatefulSet(t *testing.T) {
	c, deploy := setupFakeDeployment()
	if err := deploy.createStatefulSet(); err != nil {
		t.Fatal("failed to create statefulset", err)
	}

	nsName := types.NamespacedName{
		Name:      "storageos-statefulset",
		Namespace: defaultNS,
	}
	createdStatefulset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-statefulset",
			Namespace: defaultNS,
		},
	}
	if err := c.Get(context.Background(), nsName, createdStatefulset); err != nil {
		t.Fatal("failed to get the created object", err)
	}

	owner := createdStatefulset.GetOwnerReferences()[0]
	checkObjectOwner(t, owner, gvk)
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
			c := fake.NewFakeClient()
			deploy := NewDeployment(c, stosCluster, nil, tc.k8sVersion)
			deploy.Deploy()

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

			owner := createdDaemonset.GetOwnerReferences()[0]
			checkObjectOwner(t, owner, gvk)

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
		kubeletPluginsWatcherDriverRegArgsCount = 6
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
		Spec: api.StorageOSSpec{
			CSI: api.StorageOSCSI{
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
			c := fake.NewFakeClient()
			deploy := NewDeployment(c, stosCluster, nil, tc.k8sVersion)
			deploy.Deploy()

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

			owner := createdDaemonset.GetOwnerReferences()[0]
			checkObjectOwner(t, owner, gvk)

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
