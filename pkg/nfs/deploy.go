package nfs

import (
	"fmt"
	"strings"

	"github.com/storageos/cluster-operator/pkg/storageos"
	rbacv1 "k8s.io/api/rbac/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	appName         = "storageos"
	statefulsetKind = "statefulset"

	serviceAccountPrefix = "storageos-nfs"

	// DefaultNFSPort is the default port for NFS server.
	DefaultNFSPort = 2049
	// DefaultHTTPPort is the default port for NFS server health and metrics.
	DefaultHTTPPort = 80

	// HealthEndpointPath is the path to query on the HTTP Port for health.
	// This is hardcoded in the NFS container and not settable by the user.
	HealthEndpointPath = "/healthz"

	// VolumeFeatureReplicasKey is the label key used to set the number of
	// replicas of a StorageOS volume.
	VolumeFeatureReplicasKey = "storageos.com/replicas"

	// PodFeatureFencingKey is the label key used to enable pod fencing on a
	// pod.
	PodFeatureFencingKey = "storageos.com/fenced"

	// DefaultNFSVolumeReplicas is the default value for the NFS volume
	// replicas.
	DefaultNFSVolumeReplicas = "1"
)

var log = logf.Log.WithName("storageos.nfsserver")

// Deploy deploys a NFS server.
func (d *Deployment) Deploy() error {
	err := d.ensureService(DefaultNFSPort, DefaultHTTPPort)
	if err != nil {
		return err
	}
	if err := d.createNFSConfigMap(); err != nil {
		return err
	}

	if err := d.createServiceAccountForNFSServer(); err != nil {
		return err
	}

	// Grant OpenShift SCC permission for StatefulSet using the ClusterRole
	// created for the StorageOSCluster.
	if strings.Contains(d.cluster.Spec.K8sDistro, storageos.K8SDistroOpenShift) {
		if err := d.createClusterRoleBindingForSCC(); err != nil {
			return err
		}
	}

	// Get the NFS capacity.
	requestedCapacity := d.nfsServer.Spec.GetRequestedCapacity()
	size := &requestedCapacity

	if err := d.createStatefulSet(size, DefaultNFSPort, DefaultHTTPPort); err != nil {
		return err
	}

	status, err := d.getStatus()
	if err != nil {
		return err
	}

	if err := d.updateStatus(status); err != nil {
		log.Error(err, "Failed to update status")
	}

	return nil
}

// Due to https://github.com/kubernetes/kubernetes/issues/74916 fixed in
// 1.15, labels intended for the PVC must be set on the Pod template.
// In 1.15 and later we can just set the "app" and "nfsserver" labels here.  For
// now, pass all labels rather than check k8s versions.  The only downside is
// that the nfs pod gets storageos.com labels that don't do anything directly.
func (d *Deployment) labelsForStatefulSet(name string, labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}

	labels["app"] = appName
	labels["nfsserver"] = name

	// When fencing is enable in the cluster, set the fencing properties on the
	// NFS server pods.
	// TODO: Make the number of NFS volume replica configurable.
	if !d.cluster.Spec.DisableFencing {
		labels[PodFeatureFencingKey] = "true"
		labels[VolumeFeatureReplicasKey] = DefaultNFSVolumeReplicas
	}

	return labels
}

func (d *Deployment) createClusterRoleBindingForSCC() error {
	subjects := []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      d.getServiceAccountName(),
			Namespace: d.nfsServer.Namespace,
		},
	}
	roleRef := &rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     storageos.OpenShiftSCCClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return d.k8sResourceManager.ClusterRoleBinding(d.getClusterRoleBindingName(), subjects, roleRef).Create()
}

func (d *Deployment) getClusterRoleBindingName() string {
	return fmt.Sprintf("storageos:openshift-scc-nfs-%s", d.nfsServer.Name)
}

func (d *Deployment) getServiceAccountName() string {
	return fmt.Sprintf("%s-%s", serviceAccountPrefix, d.nfsServer.Name)
}

func (d *Deployment) createServiceAccountForNFSServer() error {
	return d.k8sResourceManager.ServiceAccount(d.getServiceAccountName(), d.nfsServer.Namespace).Create()
}
