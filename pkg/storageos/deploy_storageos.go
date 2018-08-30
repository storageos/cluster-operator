package storageos

import (
	"fmt"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	api "github.com/storageos/storageos-operator/pkg/apis/node/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	rbacv1 "k8s.io/api/rbac/v1"
	storagev1 "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/record"
)

const (
	initSecretName = "init-secret"
	tlsSecretName  = "tls-secret"
)

func deployStorageOS(m *api.StorageOS, recorder record.EventRecorder) error {
	if err := createNamespace(m); err != nil {
		return err
	}

	if err := createServiceAccountForDaemonSet(m); err != nil {
		return err
	}

	if err := createRoleForKeyMgmt(m); err != nil {
		return err
	}

	if err := createRoleBindingForKeyMgmt(m); err != nil {
		return err
	}

	if err := createInitSecret(m); err != nil {
		return err
	}

	if err := createDaemonSet(m); err != nil {
		return err
	}

	if err := createService(m); err != nil {
		return err
	}

	if m.Spec.Ingress.Enabled {
		if m.Spec.Ingress.TLS {
			if err := createTLSSecret(m); err != nil {
				return err
			}
		}

		if err := createIngress(m); err != nil {
			return err
		}
	}

	if m.Spec.EnableCSI {
		// Create CSI exclusive resources.
		if err := createClusterRoleForDriverRegistrar(m); err != nil {
			return err
		}

		if err := createClusterRoleBindingForDriverRegistrar(m); err != nil {
			return err
		}

		if err := createServiceAccountForStatefulSet(m); err != nil {
			return err
		}

		if err := createClusterRoleForProvisioner(m); err != nil {
			return err
		}

		if err := createClusterRoleForAttacher(m); err != nil {
			return err
		}

		if err := createClusterRoleBindingForProvisioner(m); err != nil {
			return err
		}

		if err := createClusterRoleBindingForAttacher(m); err != nil {
			return err
		}

		if err := createStatefulSet(m); err != nil {
			return err
		}
	}

	if err := createStorageClass(m); err != nil {
		return err
	}

	status, err := getStorageOSStatus(m)
	if err != nil {
		return fmt.Errorf("failed to get storageos status: %v", err)
	}
	return updateStorageOSStatus(m, status, recorder)
}

func createNamespace(m *api.StorageOS) error {
	ns := &v1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: m.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
	}

	addOwnerRefToObject(ns, asOwner(m))
	if err := sdk.Create(ns); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create namespace: %v", err)
	}
	return nil
}

func createServiceAccountForDaemonSet(m *api.StorageOS) error {
	sa := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-daemonset-sa",
			Namespace: m.Spec.GetResourceNS(),
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

func createServiceAccountForStatefulSet(m *api.StorageOS) error {
	sa := &v1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-statefulset-sa",
			Namespace: m.Spec.GetResourceNS(),
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
			Namespace: m.Spec.GetResourceNS(),
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

func createClusterRoleForDriverRegistrar(m *api.StorageOS) error {
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "driver-registrar-role",
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Rules: []rbacv1.PolicyRule{
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
		},
	}

	addOwnerRefToObject(role, asOwner(m))
	if err := sdk.Create(role); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role: %v", err)
	}
	return nil
}

func createClusterRoleForProvisioner(m *api.StorageOS) error {
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "csi-provisioner-role",
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"list", "watch", "create", "delete"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumeclaims"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{"storageo.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"list", "watch", "get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"list", "watch", "create", "update", "patch"},
			},
		},
	}

	addOwnerRefToObject(role, asOwner(m))
	if err := sdk.Create(role); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role: %v", err)
	}
	return nil
}

func createClusterRoleForAttacher(m *api.StorageOS) error {
	role := &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "csi-attacher-role",
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{""},
				Resources: []string{"persistentvolumes"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"nodes"},
				Verbs:     []string{"get", "list", "watch"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"storageclasses"},
				Verbs:     []string{"list", "watch", "get"},
			},
			{
				APIGroups: []string{"storage.k8s.io"},
				Resources: []string{"volumeattachments"},
				Verbs:     []string{"get", "list", "watch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"events"},
				Verbs:     []string{"list", "watch", "create", "update", "patch"},
			},
		},
	}

	addOwnerRefToObject(role, asOwner(m))
	if err := sdk.Create(role); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role: %v", err)
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
			Namespace: m.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "storageos-daemonset-sa",
				Namespace: m.Spec.GetResourceNS(),
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

func createClusterRoleBindingForDriverRegistrar(m *api.StorageOS) error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "driver-registrar-binding",
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "storageos-daemonset-sa",
				Namespace: m.Spec.GetResourceNS(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "driver-registrar-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	addOwnerRefToObject(roleBinding, asOwner(m))
	if err := sdk.Create(roleBinding); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role binding: %v", err)
	}
	return nil
}

func createClusterRoleBindingForProvisioner(m *api.StorageOS) error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "csi-provisioner-binding",
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "storageos-statefulset-sa",
				Namespace: m.Spec.GetResourceNS(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "csi-provisioner-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	addOwnerRefToObject(roleBinding, asOwner(m))
	if err := sdk.Create(roleBinding); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role binding: %v", err)
	}
	return nil
}

func createClusterRoleBindingForAttacher(m *api.StorageOS) error {
	roleBinding := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "csi-attacher-binding",
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      "storageos-statefulset-sa",
				Namespace: m.Spec.GetResourceNS(),
			},
		},
		RoleRef: rbacv1.RoleRef{
			Kind:     "ClusterRole",
			Name:     "csi-attacher-role",
			APIGroup: "rbac.authorization.k8s.io",
		},
	}

	addOwnerRefToObject(roleBinding, asOwner(m))
	if err := sdk.Create(roleBinding); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cluster role binding: %v", err)
	}
	return nil
}

func createDaemonSet(m *api.StorageOS) error {
	ls := labelsForDaemonSet(m.Name)
	privileged := true
	mountPropagationBidirectional := v1.MountPropagationBidirectional
	hostpathDirOrCreate := v1.HostPathDirectoryOrCreate
	hostpathDir := v1.HostPathDirectory
	allowPrivilegeEscalation := true

	dset := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Spec.GetResourceNS(),
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
									MountPropagation: &mountPropagationBidirectional,
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
							Ports: []v1.ContainerPort{{
								ContainerPort: 5705,
								Name:          "api",
							}},
							LivenessProbe: &v1.Probe{
								InitialDelaySeconds: int32(65),
								TimeoutSeconds:      int32(10),
								FailureThreshold:    int32(5),
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/v1/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
									},
								},
							},
							ReadinessProbe: &v1.Probe{
								InitialDelaySeconds: int32(65),
								TimeoutSeconds:      int32(10),
								FailureThreshold:    int32(5),
								Handler: v1.Handler{
									HTTPGet: &v1.HTTPGetAction{
										Path: "/v1/health",
										Port: intstr.IntOrString{Type: intstr.String, StrVal: "api"},
									},
								},
							},
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
									Name: "ADMIN_USERNAME",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: initSecretName,
											},
											Key: "username",
										},
									},
								},
								{
									Name: "ADMIN_PASSWORD",
									ValueFrom: &v1.EnvVarSource{
										SecretKeyRef: &v1.SecretKeySelector{
											LocalObjectReference: v1.LocalObjectReference{
												Name: initSecretName,
											},
											Key: "password",
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
									Value: m.Spec.GetResourceNS(),
								},
							},
							SecurityContext: &v1.SecurityContext{
								Privileged: &privileged,
								Capabilities: &v1.Capabilities{
									Add: []v1.Capability{"SYS_ADMIN"},
								},
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
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
									Name:             "state",
									MountPath:        "/var/lib/storageos",
									MountPropagation: &mountPropagationBidirectional,
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
						// TODO: Add sharedDir volume.
					},
				},
			},
		},
	}

	// If kubelet is running in a container, sharedDir should be set.
	if m.Spec.SharedDir != "" {
		envVar := v1.EnvVar{
			Name:  "DEVICE_DIR",
			Value: fmt.Sprintf("%s/devices", m.Spec.SharedDir),
		}
		dset.Spec.Template.Spec.Containers[0].Env = append(dset.Spec.Template.Spec.Containers[0].Env, envVar)

		sharedDir := v1.Volume{
			Name: "shared",
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: m.Spec.SharedDir,
				},
			},
		}
		dset.Spec.Template.Spec.Volumes = append(dset.Spec.Template.Spec.Volumes, sharedDir)

		volMnt := v1.VolumeMount{
			Name:             "shared",
			MountPath:        m.Spec.SharedDir,
			MountPropagation: &mountPropagationBidirectional,
		}
		dset.Spec.Template.Spec.Containers[0].VolumeMounts = append(dset.Spec.Template.Spec.Containers[0].VolumeMounts, volMnt)
	}

	// Add CSI specific configurations if enabled.
	if m.Spec.EnableCSI {
		vols := []v1.Volume{
			{
				Name: "registrar-socket-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/var/lib/kubelet/device-plugins/",
						Type: &hostpathDirOrCreate,
					},
				},
			},
			{
				Name: "kubelet-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/var/lib/kubelet",
						Type: &hostpathDir,
					},
				},
			},
			{
				Name: "plugin-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/var/lib/kubelet/plugins/storageos/",
						Type: &hostpathDirOrCreate,
					},
				},
			},
			{
				Name: "device-dir",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: "/dev",
						Type: &hostpathDir,
					},
				},
			},
		}

		dset.Spec.Template.Spec.Volumes = append(dset.Spec.Template.Spec.Volumes, vols...)

		volMnts := []v1.VolumeMount{
			{
				Name:             "kubelet-dir",
				MountPath:        "/var/lib/kubelet",
				MountPropagation: &mountPropagationBidirectional,
			},
			{
				Name:      "plugin-dir",
				MountPath: "/var/lib/kubelet/plugins/storageos/",
			},
			{
				Name:      "device-dir",
				MountPath: "/dev",
			},
		}

		// Append volume mounts to the first container, the only container is the node container, at this point.
		dset.Spec.Template.Spec.Containers[0].VolumeMounts = append(dset.Spec.Template.Spec.Containers[0].VolumeMounts, volMnts...)

		envVar := []v1.EnvVar{
			{
				Name:  "CSI_ENDPOINT",
				Value: "unix://var/lib/kubelet/plugins/storageos/csi.sock",
			},
		}
		// Append env vars to the first container, node container.
		dset.Spec.Template.Spec.Containers[0].Env = append(dset.Spec.Template.Spec.Containers[0].Env, envVar...)

		driverReg := v1.Container{
			Image:           "quay.io/k8scsi/driver-registrar:v0.2.0",
			Name:            "csi-driver-registrar",
			ImagePullPolicy: v1.PullIfNotPresent,
			Args: []string{
				"--v=5",
				"--csi-address=$(ADDRESS)",
			},
			Env: []v1.EnvVar{
				{
					Name:  "ADDRESS",
					Value: "/csi/csi.sock",
				},
				{
					Name: "KUBE_NODE_NAME",
					ValueFrom: &v1.EnvVarSource{
						FieldRef: &v1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
			},
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "plugin-dir",
					MountPath: "/csi",
				},
				{
					Name:      "registrar-socket-dir",
					MountPath: "/var/lib/csi/sockets/",
				},
			},
		}
		dset.Spec.Template.Spec.Containers = append(dset.Spec.Template.Spec.Containers, driverReg)
	}

	addOwnerRefToObject(dset, asOwner(m))
	if err := sdk.Create(dset); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create daemonset: %v", err)
	}
	return nil
}

func createStatefulSet(m *api.StorageOS) error {
	ls := labelsForStatefulSet(m.Name)
	replicas := int32(1)
	hostpathDirOrCreate := v1.HostPathDirectoryOrCreate

	sset := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-statefulset",
			Namespace: m.Spec.GetResourceNS(),
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: "storageos",
			Replicas:    &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: ls,
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: ls,
				},
				Spec: v1.PodSpec{
					ServiceAccountName: "storageos-statefulset-sa",
					Containers: []v1.Container{
						{
							Image:           "quay.io/k8scsi/csi-provisioner:canary",
							Name:            "csi-external-provisioner",
							ImagePullPolicy: v1.PullIfNotPresent,
							Args: []string{
								"--v=5",
								"--provisioner=storageos",
								"--csi-address=$(ADDRESS)",
							},
							Env: []v1.EnvVar{
								{
									Name:  "ADDRESS",
									Value: "/csi/csi.sock",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "plugin-dir",
									MountPath: "/csi",
								},
							},
						},
						{
							Image:           "quay.io/k8scsi/csi-attacher:canary",
							Name:            "csi-external-attacher",
							ImagePullPolicy: v1.PullIfNotPresent,
							Args: []string{
								"--v=5",
								"--csi-address=$(ADDRESS)",
							},
							Env: []v1.EnvVar{
								{
									Name:  "ADDRESS",
									Value: "/csi/csi.sock",
								},
							},
							VolumeMounts: []v1.VolumeMount{
								{
									Name:      "plugin-dir",
									MountPath: "/csi",
								},
							},
						},
					},
					Volumes: []v1.Volume{
						{
							Name: "plugin-dir",
							VolumeSource: v1.VolumeSource{
								HostPath: &v1.HostPathVolumeSource{
									Path: "/var/lib/kubelet/plugins/storageos/",
									Type: &hostpathDirOrCreate,
								},
							},
						},
					},
				},
			},
		},
	}

	addOwnerRefToObject(sset, asOwner(m))
	if err := sdk.Create(sset); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create statefulset: %v", err)
	}
	return nil
}

func createService(m *api.StorageOS) error {
	svc := &v1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Spec.Service.Name,
			Namespace: m.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
			Annotations: m.Spec.Service.Annotations,
		},
		Spec: v1.ServiceSpec{
			Type: v1.ServiceType(m.Spec.Service.Type),
			Ports: []v1.ServicePort{
				{
					Name:       m.Spec.Service.Name,
					Protocol:   "TCP",
					Port:       int32(m.Spec.Service.InternalPort),
					TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(m.Spec.Service.ExternalPort)},
				},
			},
			Selector: map[string]string{
				"app":  "storageos",
				"kind": "daemonset",
			},
		},
	}

	addOwnerRefToObject(svc, asOwner(m))
	if err := sdk.Create(svc); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create service: %v", err)
	}

	// Patch storageos-api secret with above service IP in apiAddress.
	if !m.Spec.EnableCSI {
		secret := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.Spec.SecretRefName,
				Namespace: m.Spec.SecretRefNamespace,
			},
		}
		if err := sdk.Get(secret); err != nil {
			return err
		}

		if err := sdk.Get(svc); err != nil {
			return err
		}

		apiAddress := fmt.Sprintf("tcp://%s:5705", svc.Spec.ClusterIP)
		secret.Data["apiAddress"] = []byte(apiAddress)

		if err := sdk.Update(secret); err != nil {
			return err
		}
	}

	return nil
}

func createIngress(m *api.StorageOS) error {
	ingress := &v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "extensions/v1beta1",
			Kind:       "Ingress",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "storageos-ingress",
			Namespace: m.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
			Annotations: m.Spec.Ingress.Annotations,
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: m.Spec.Service.Name,
				ServicePort: intstr.IntOrString{Type: intstr.Int, IntVal: int32(m.Spec.Service.ExternalPort)},
			},
		},
	}

	if m.Spec.Ingress.TLS {
		ingress.Spec.TLS = []v1beta1.IngressTLS{
			v1beta1.IngressTLS{
				Hosts:      []string{m.Spec.Ingress.Hostname},
				SecretName: tlsSecretName,
			},
		}
	}

	addOwnerRefToObject(ingress, asOwner(m))
	if err := sdk.Create(ingress); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create ingress")
	}
	return nil
}

func createTLSSecret(m *api.StorageOS) error {
	cert, key, err := getTLSData(m)
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      tlsSecretName,
			Namespace: m.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Type: v1.SecretType("kubernetes.io/tls"),
		Data: map[string][]byte{
			"tls.crt": cert,
			"tls.key": key,
		},
	}

	addOwnerRefToObject(secret, asOwner(m))
	if err := sdk.Create(secret); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create tls-secret: %v", err)
	}
	return nil
}

func createInitSecret(m *api.StorageOS) error {
	username, password, err := getAdminCreds(m)
	if err != nil {
		return err
	}

	secret := &v1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      initSecretName,
			Namespace: m.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Type: v1.SecretType("kubernetes.io/storageos"),
		Data: map[string][]byte{
			"username": username,
			"password": password,
		},
	}

	addOwnerRefToObject(secret, asOwner(m))
	if err := sdk.Create(secret); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create init secret: %v", err)
	}
	return nil
}

func getAdminCreds(m *api.StorageOS) ([]byte, []byte, error) {
	var username, password []byte
	if m.Spec.SecretRefName != "" && m.Spec.SecretRefNamespace != "" {
		se := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.Spec.SecretRefName,
				Namespace: m.Spec.SecretRefNamespace,
			},
		}
		err := sdk.Get(se)
		if err != nil {
			return nil, nil, err
		}

		username = se.Data["apiUsername"]
		password = se.Data["apiPassword"]
	} else {
		// Use the default credentials.
		username = []byte("storageos")
		password = []byte("storageos")
	}

	return username, password, nil
}

func getTLSData(m *api.StorageOS) ([]byte, []byte, error) {
	var cert, key []byte
	if m.Spec.SecretRefName != "" && m.Spec.SecretRefNamespace != "" {
		se := &v1.Secret{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Secret",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      m.Spec.SecretRefName,
				Namespace: m.Spec.SecretRefNamespace,
			},
		}
		err := sdk.Get(se)
		if err != nil {
			return nil, nil, err
		}

		cert = se.Data["tls.crt"]
		key = se.Data["tls.key"]
	} else {
		cert = []byte("")
		key = []byte("")
	}

	return cert, key, nil
}

func createStorageClass(m *api.StorageOS) error {
	// Provisioner name for in-tree storage plugin.
	provisioner := "kubernetes.io/storageos"

	if m.Spec.EnableCSI {
		provisioner = "storageos"
	}

	sc := &storagev1.StorageClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "storage.k8s.io/v1",
			Kind:       "StorageClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "fast",
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Provisioner: provisioner,
		Parameters: map[string]string{
			"pool":   "default",
			"fsType": "ext4",
		},
	}

	if m.Spec.EnableCSI {
		// Add CSI creds secrets in parameters.
	} else {
		// Add StorageOS admin secrets name and namespace.
		sc.Parameters["adminSecretNamespace"] = m.Spec.SecretRefNamespace
		sc.Parameters["adminSecretName"] = m.Spec.SecretRefName
	}

	addOwnerRefToObject(sc, asOwner(m))
	if err := sdk.Create(sc); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create storage class: %v", err)
	}
	return nil
}

func labelsForDaemonSet(name string) map[string]string {
	return map[string]string{"app": "storageos", "storageos_cr": name, "kind": "daemonset"}
}

func labelsForStatefulSet(name string) map[string]string {
	return map[string]string{"app": "storageos", "storageos_cr": name, "kind": "statefulset"}
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

func nodeList() *v1.NodeList {
	return &v1.NodeList{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Node",
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

func getNodeIPs(nodes []v1.Node) []string {
	var ips []string
	for _, node := range nodes {
		ips = append(ips, node.Status.Addresses[0].Address)
	}
	return ips
}
