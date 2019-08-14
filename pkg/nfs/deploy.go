package nfs

import (
	"fmt"
	"strings"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/pkg/util"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	appName         = "storageos"
	statefulsetKind = "statefulset"

	serviceAccountPrefix = "storageos-nfs"

	DefaultNFSPort     = 2049
	DefaultMetricsPort = 9587
)

var log = logf.Log.WithName("storageos.nfsserver")

// Deploy deploys a NFS server.
func (d *Deployment) Deploy() error {
	// Get the current StorageOS cluster.
	stosClusters, err := d.stosClient.StorageosV1().StorageOSClusters("").List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var currentCluster storageosv1.StorageOSCluster

	for _, cluster := range stosClusters.Items {
		// Only one cluster can be in running phase at a time.
		if cluster.Status.Phase == storageosv1.ClusterPhaseRunning {
			currentCluster = cluster
			break
		}
	}

	d.cluster = &currentCluster

	// Update NFSServer spec StorageClassName value.
	d.nfsServer.Spec.StorageClassName = d.nfsServer.Spec.GetStorageClassName(d.cluster.Spec.GetStorageClassName())

	err = d.ensureService(DefaultNFSPort, DefaultMetricsPort)
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
	if strings.Contains(currentCluster.Spec.K8sDistro, storageos.K8SDistroOpenShift) {
		if err := d.createClusterRoleBindingForSCC(); err != nil {
			return err
		}
	}

	// Get the NFS capacity.
	requestedCapacity := d.nfsServer.Spec.GetRequestedCapacity()
	size := &requestedCapacity

	if err := d.createStatefulSet(size, DefaultNFSPort, DefaultMetricsPort); err != nil {
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

	if !d.cluster.Spec.DisableFencing {
		labels["storageos.com/fenced"] = "true"
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
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     storageos.OpenShiftSCCClusterRoleName,
		APIGroup: "rbac.authorization.k8s.io",
	}
	return util.CreateClusterRoleBinding(d.client, d.getClusterRoleBindingName(), subjects, roleRef)
}

func (d *Deployment) getClusterRoleBindingName() string {
	return fmt.Sprintf("storageos:openshift-scc-nfs-%s", d.nfsServer.Name)
}

func (d *Deployment) getServiceAccountName() string {
	return fmt.Sprintf("%s-%s", serviceAccountPrefix, d.nfsServer.Name)
}

func (d *Deployment) createServiceAccountForNFSServer() error {
	return util.CreateServiceAccount(d.client, d.getServiceAccountName(), d.nfsServer.Namespace)
}
