package k8s

import (
	"context"
	"testing"

	"github.com/storageos/cluster-operator/pkg/util/k8s/resource"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	extensionsv1beta1 "k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestResourceManager tests ResourceManager and the resources in the
// k8s/resource package.
// TODO: Get() method of the Resource(s) are not tested here. Need to find a
// better way to test it.
func TestResourceManager(t *testing.T) {
	// NamespacedName for all the resources.
	nsName := types.NamespacedName{
		Name:      "SomeName",
		Namespace: "SomeNamespace",
	}

	testcases := []struct {
		name         string
		create       func(*ResourceManager, types.NamespacedName) error
		delete       func(*ResourceManager, types.NamespacedName) error
		wantResource runtime.Object
	}{
		{
			name: resource.ConfigMapKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ConfigMap(nsName.Name, nsName.Namespace, nil).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ConfigMap(nsName.Name, nsName.Namespace, nil).Delete()
			},
			wantResource: &corev1.ConfigMap{},
		},
		{
			name: resource.DaemonSetKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.DaemonSet(nsName.Name, nsName.Namespace, &appsv1.DaemonSetSpec{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.DaemonSet(nsName.Name, nsName.Namespace, nil).Delete()
			},
			wantResource: &appsv1.DaemonSet{},
		},
		{
			name: resource.DeploymentKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Deployment(nsName.Name, nsName.Namespace, &appsv1.DeploymentSpec{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Deployment(nsName.Name, nsName.Namespace, nil).Delete()
			},
			wantResource: &appsv1.Deployment{},
		},
		{
			name: resource.IngressKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Ingress(nsName.Name, nsName.Namespace, nil, &extensionsv1beta1.IngressSpec{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Ingress(nsName.Name, nsName.Namespace, nil, nil).Delete()
			},
			wantResource: &extensionsv1beta1.Ingress{},
		},
		{
			name: resource.ServiceAccountKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ServiceAccount(nsName.Name, nsName.Namespace).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ServiceAccount(nsName.Name, nsName.Namespace).Delete()
			},
			wantResource: &corev1.ServiceAccount{},
		},
		{
			name: resource.RoleKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Role(nsName.Name, nsName.Namespace, []rbacv1.PolicyRule{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Role(nsName.Name, nsName.Namespace, nil).Delete()
			},
			wantResource: &rbacv1.Role{},
		},
		{
			name: resource.RoleBindingKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.RoleBinding(nsName.Name, nsName.Namespace, nil, &rbacv1.RoleRef{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.RoleBinding(nsName.Name, nsName.Namespace, nil, nil).Delete()
			},
			wantResource: &rbacv1.RoleBinding{},
		},
		{
			name: resource.ClusterRoleKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ClusterRole(nsName.Name, []rbacv1.PolicyRule{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ClusterRole(nsName.Name, nil).Delete()
			},
			wantResource: &rbacv1.ClusterRole{},
		},
		{
			name: resource.ClusterRoleBindingKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ClusterRoleBinding(nsName.Name, nil, &rbacv1.RoleRef{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.ClusterRoleBinding(nsName.Name, nil, nil).Delete()
			},
			wantResource: &rbacv1.ClusterRoleBinding{},
		},
		{
			name: resource.SecretKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Secret(nsName.Name, nsName.Namespace, corev1.SecretTypeOpaque, map[string][]byte{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Secret(nsName.Name, nsName.Namespace, corev1.SecretTypeOpaque, nil).Delete()
			},
			wantResource: &corev1.Secret{},
		},
		{
			name: resource.ServiceKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Service(nsName.Name, nsName.Namespace, map[string]string{}, map[string]string{}, &corev1.ServiceSpec{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.Service(nsName.Name, nsName.Namespace, nil, nil, nil).Delete()
			},
			wantResource: &corev1.Service{},
		},
		{
			name: resource.StatefulSetKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.StatefulSet(nsName.Name, nsName.Namespace, &appsv1.StatefulSetSpec{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.StatefulSet(nsName.Name, nsName.Namespace, nil).Delete()
			},
			wantResource: &appsv1.StatefulSet{},
		},
		{
			name: resource.StorageClassKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.StorageClass(nsName.Name, "storageos", map[string]string{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.StorageClass(nsName.Name, "storageos", nil).Delete()
			},
			wantResource: &storagev1.StorageClass{},
		},
		{
			name: resource.PVCKind,
			create: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.PersistentVolumeClaim(nsName.Name, nsName.Namespace, &corev1.PersistentVolumeClaimSpec{}).Create()
			},
			delete: func(rm *ResourceManager, nsName types.NamespacedName) error {
				return rm.PersistentVolumeClaim(nsName.Name, nsName.Namespace, nil).Delete()
			},
			wantResource: &corev1.PersistentVolumeClaim{},
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			client := fake.NewFakeClient()

			labels := map[string]string{"app": "testapp"}
			rm := NewResourceManager(client).SetLabels(labels)

			// Create resource.
			if err := tc.create(rm, nsName); err != nil {
				t.Errorf("failed to create %s: %v", tc.name, err)
			}

			switch ty := tc.wantResource.(type) {
			case *rbacv1.ClusterRole, *rbacv1.ClusterRoleBinding:
				nsName.Namespace = ""
				// Workaround to avoid unused variable.
				_ = ty
			default:
			}

			if err := client.Get(context.TODO(), nsName, tc.wantResource); err != nil {
				t.Errorf("expected %s to be created but not found: %v", tc.name, err)
			}

			if err := tc.delete(rm, nsName); err != nil {
				t.Errorf("failed to delete %s: %v", tc.name, err)
			}

			// Delete the resource with name and namespace reference, and ensure its
			// deleted.
			if err := client.Get(context.TODO(), nsName, tc.wantResource); err != nil {
				if !apierrors.IsNotFound(err) {
					t.Errorf("expected error to be NotFound, but got: %v", err)
				}
			} else {
				t.Errorf("expected %s to not exist", tc.name)
			}
		})
	}
}
