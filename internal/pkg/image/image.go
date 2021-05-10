package image

import "os"

// Default image constant variables.
const (
	DefaultNodeContainerImage = "storageos/node:v2.4.0-rc.1"
	DefaultInitContainerImage = "storageos/init:v2.1.0"
	DefaultAPIManagerImage    = "storageos/api-manager:v1.1.1"

	CSINodeDriverRegistrarContainerImage = "quay.io/k8scsi/csi-node-driver-registrar:v2.1.0"
	CSIExternalProvisionerContainerImage = "storageos/csi-provisioner:v2.1.1-patched"
	CSIExternalAttacherContainerImage    = "quay.io/k8scsi/csi-attacher:v3.1.0"
	CSIExternalResizerContainerImage     = "quay.io/k8scsi/csi-resizer:v1.1.0"
	CSILivenessProbeContainerImage       = "quay.io/k8scsi/livenessprobe:v2.2.0"

	DefaultKubeSchedulerContainerRegistry = "k8s.gcr.io/kube-scheduler"
)

// Environment variables for setting default images.
const (
	StorageOSNodeImageEnvVar = "RELATED_IMAGE_STORAGEOS_NODE"
	StorageOSInitImageEnvVar = "RELATED_IMAGE_STORAGEOS_INIT"

	CSINodeDriverRegistrarImageEnvVar      = "RELATED_IMAGE_CSI_NODE_DRIVER_REGISTRAR"
	CSIExternalProvisionerImageEnvVar      = "RELATED_IMAGE_CSI_EXTERNAL_PROVISIONER"
	CSIExternalAttacherImageEnvVar         = "RELATED_IMAGE_CSI_EXTERNAL_ATTACHER"
	CSIExternalResizerContainerImageEnvVar = "RELATED_IMAGE_CSI_EXTERNAL_RESIZER"
	CSILivenessProbeImageEnvVar            = "RELATED_IMAGE_CSI_LIVENESS_PROBE"

	KubeSchedulerImageEnvVar = "RELATED_IMAGE_KUBE_SCHEDULER"
	APIManagerEnvVar         = "RELATED_IMAGE_API_MANAGER"
)

// GetDefaultImage checks the environment variable for an image. If not found,
// it returns a default image.
func GetDefaultImage(imageEnvVar, defaultImage string) string {
	img := os.Getenv(imageEnvVar)
	if img != "" {
		return img
	}
	return defaultImage
}
