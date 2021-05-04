package image

import "os"

// Default image constant variables.
const (
	DefaultNodeContainerImage                 = "storageos/node:v2.4.0-rc.1"
	DefaultInitContainerImage                 = "storageos/init:v2.1.0"
	CSIv1ClusterDriverRegistrarContainerImage = "quay.io/k8scsi/csi-cluster-driver-registrar:v1.0.1"
	CSIv1NodeDriverRegistrarContainerImage    = "quay.io/k8scsi/csi-node-driver-registrar:v2.1.0"
	CSIv1ExternalProvisionerContainerImageV2  = "storageos/csi-provisioner:v2.1.1-patched"
	CSIv1ExternalAttacherContainerImage       = "quay.io/k8scsi/csi-attacher:v1.2.1"
	CSIv1ExternalAttacherContainerImageV2     = "quay.io/k8scsi/csi-attacher:v2.2.0"
	CSIv1ExternalAttacherContainerImageV3     = "quay.io/k8scsi/csi-attacher:v3.1.0"
	CSIv1ExternalResizerContainerImage        = "quay.io/k8scsi/csi-resizer:v1.1.0"
	CSIv1LivenessProbeContainerImage          = "quay.io/k8scsi/livenessprobe:v2.2.0"
	CSIv0DriverRegistrarContainerImage        = "quay.io/k8scsi/driver-registrar:v0.4.2"
	CSIv0ExternalProvisionerContainerImage    = "storageos/csi-provisioner:v0.4.3"
	CSIv0ExternalAttacherContainerImage       = "quay.io/k8scsi/csi-attacher:v0.4.2"
	DefaultNFSContainerImage                  = "storageos/nfs:1.0.0"
	DefaultAPIManagerImage                    = "storageos/api-manager:v1.1.1"

	DefaultHyperkubeContainerRegistry = "gcr.io/google_containers/hyperkube"

	DefaultKubeSchedulerContainerRegistry = "k8s.gcr.io/kube-scheduler"
)

// Environment variables for setting default images.
const (
	StorageOSNodeImageEnvVar = "RELATED_IMAGE_STORAGEOS_NODE"
	StorageOSInitImageEnvVar = "RELATED_IMAGE_STORAGEOS_INIT"

	CSIv1ClusterDriverRegistrarImageEnvVar   = "RELATED_IMAGE_CSIV1_CLUSTER_DRIVER_REGISTRAR"
	CSIv1NodeDriverRegistrarImageEnvVar      = "RELATED_IMAGE_CSIV1_NODE_DRIVER_REGISTRAR"
	CSIv1ExternalProvisionerImageEnvVar      = "RELATED_IMAGE_CSIV1_EXTERNAL_PROVISIONER"
	CSIv1ExternalAttacherImageEnvVar         = "RELATED_IMAGE_CSIV1_EXTERNAL_ATTACHER"
	CSIv1ExternalAttacherv2ImageEnvVar       = "RELATED_IMAGE_CSIV1_EXTERNAL_ATTACHER_V2"
	CSIv1ExternalAttacherv3ImageEnvVar       = "RELATED_IMAGE_CSIV1_EXTERNAL_ATTACHER_V3"
	CSIv1ExternalResizerContainerImageEnvVar = "RELATED_IMAGE_CSIV1_EXTERNAL_RESIZER"
	CSIv1LivenessProbeImageEnvVar            = "RELATED_IMAGE_CSIV1_LIVENESS_PROBE"

	CSIv0DriverRegistrarImageEnvVar     = "RELATED_IMAGE_CSIV0_DRIVER_REGISTRAR"
	CSIv0ExternalProvisionerImageEnvVar = "RELATED_IMAGE_CSIV0_EXTERNAL_PROVISIONER"
	CSIv0ExternalAttacherImageEnvVar    = "RELATED_IMAGE_CSIV0_EXTERNAL_ATTACHER"

	NFSImageEnvVar           = "RELATED_IMAGE_NFS"
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
