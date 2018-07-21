package storageos

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/storageos/storageos-operator/pkg/apis/node/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func deployStorageOS(m *api.StorageOS) error {
	if err := createServiceAccountForDaemonSet(m); err != nil {
		return err
	}

	if err := createRoleForKeyMgmt(m); err != nil {
		return err
	}

	if err := createRoleBindingForKeyMgmt(m); err != nil {
		return err
	}

	if err := createDaemonSet(m); err != nil {
		return err
	}

	status, err := getStorageOSStatus(m)
	if err != nil {
		return fmt.Errorf("failed to get storageos status: %v", err)
	}
	return updateStorageOSStatus(m, status)
}

func createServiceAccountForDaemonSet(m *api.StorageOS) error {
	sa := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-daemonset-sa",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app": "storageos",
			},
		},
	}

	addOwnerRefToObject(sa, asOwner(m))
	if err := sdk.Create(sa); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service account: %v", err)
	}
	return nil
}

func createRoleForKeyMgmt(m *api.StorageOS) error {
	role := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "key-management-role",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get", "list", "create", "delete"},
			},
		},
	}

	addOwnerRefToObject(role, asOwner(m))
	if err := sdk.Create(role); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create role: %v", err)
	}
	return nil
}

func createRoleBindingForKeyMgmt(m *api.StorageOS) error {
	roleBinding := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "key-management-binding",
			Namespace: m.Namespace,
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "storageos-daemonset-sa",
				Namespace: m.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "Role",
			Name:     "key-management-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	addOwnerRefToObject(roleBinding, asOwner(m))
	if err := sdk.Create(roleBinding); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create role binding: %v", err)
	}
	return nil
}

func createDaemonSet(m *api.StorageOS) error {
	ls := labelsForStorageOS(m.Name)
	privileged := true
	mountPropagation := v1.MountPropagationBidirectional

	dset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "storageos-daemonset-sa",
					HostPID:            true,
					HostNetwork:        true,
					InitContainers: []v1.Container{
						{
							Name:  "enable-lio",
							Image: "storageos/init:0.1",
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "kernel-modules",
									MountPath: "/lib/modules",
									ReadOnly:  true,
								},
								{
									Name:             "sys",
									MountPath:        "/sys",
									MountPropagation: &mountPropagation,
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &v1.Capabilities{
									Add: []v1.Capability{"SYS_ADMIN"},
								},
							},
						},
					},
					Containers: []v1.Container{
						{
							Image: "storageos/node:1.0.0-rc4",
							Name:  "storageos",
							Args:  []string{"server"},
							// Command: []string{"storageos", "server"},
							Ports: []v1.ContainerPort{{
								ContainerPort: 5705,
								Name:          "api",
							}},
							Env: []v1.EnvVar{
								{
									Name: "HOSTNAME",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "spec.nodeName",
										},
									},
								},
								{
									Name:  "JOIN",
									Value: m.Spec.Join,
									// ValueFrom: &v1.EnvVarSource{
									// 	FieldRef: &v1.ObjectFieldSelector{
									// 		FieldPath: "status.podIP",
									// 	},
									// },
								},
								{
									Name: "ADVERTISE_IP",
									ValueFrom: &v1.EnvVarSource{
										FieldRef: &v1.ObjectFieldSelector{
											FieldPath: "status.podIP",
										},
									},
								},
								{
									Name:  "NAMESPACE",
									Value: m.Namespace,
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &v1.Capabilities{
									Add: []v1.Capability{"SYS_ADMIN"},
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "fuse",
									MountPath: "/dev/fuse",
								},
								{
									Name:      "sys",
									MountPath: "/sys",
								},
								{
									Name:      "state",
									MountPath: "/var/lib/storageos",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "kernel-modules",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/lib/modules",
								},
							},
						},
						{
							Name: "fuse",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/dev/fuse",
								},
							},
						},
						{
							Name: "sys",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
						{
							Name: "state",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/storageos",
								},
							},
						},
					},
				},
			},
		},
	}

	addOwnerRefToObject(dset, asOwner(m))
	if err := sdk.Create(dset); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create daemonset: %v", err)
	}
	return nil
}

func labelsForStorageOS(name string) map[string]string {
	return map[string]string{"app": "storageos", "storageos_cr": name}
}

func addOwnerRefToObject(obj metav1.Object, ownerRef metav1.OwnerReference) {
	obj.SetOwnerReferences(append(obj.GetOwnerReferences(), ownerRef))
}

func asOwner(m *api.StorageOS) metav1.OwnerReference {
	trueVar := true
	return metav1.OwnerReference{
		APIVersion: m.APIVersion,
		Kind:       m.Kind,
		Name:       m.Name,
		UID:        m.UID,
		Controller: &trueVar,
	}
}

func podList() *v1.PodList {
	return &v1.PodList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
	}
}

func getPodNames(pods []v1.Pod) []string {
	var podNames []string
	for _, pod := range pods {
		podNames = append(podNames, pod.Name)
	}
	return podNames
}

func getNodeNames(pods []v1.Pod) []string {
	var nodes []string
	for _, pod := range pods {
		nodes = append(nodes, pod.Spec.NodeName)
	}
	return nodes
}
