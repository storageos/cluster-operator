package nfs

import (
	"fmt"
	"strings"

	"github.com/operator-framework/operator-sdk/pkg/metrics"
	"github.com/storageos/cluster-operator/pkg/storageos"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
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

	pvcVS := d.nfsServer.Spec.PersistentVolumeClaim

	// If no existing PVC Volume Source is specified in the spec, create a new
	// PVC with NFS Server name.
	if pvcVS.ClaimName == "" {
		// Create a PVC with the same name as the NFS Server.
		if err := d.createPVC(size); err != nil {
			return err
		}
		pvcVS = corev1.PersistentVolumeClaimVolumeSource{
			ClaimName: d.nfsServer.Name,
		}
	}

	// Create a StatefulSet NFS Server with PVC Volume Source.
	if err := d.createStatefulSet(&pvcVS, DefaultNFSPort, DefaultHTTPPort); err != nil {
		return err
	}

	status, err := d.getStatus()
	if err != nil {
		return err
	}

	if err := d.updateStatus(status); err != nil {
		log.Error(err, "Failed to update status")
	}

	if err := d.createServiceMonitor(); err != nil {
		// Ignore if the ServiceMonitor already exists.
		if !errors.IsAlreadyExists(err) {
			log.Error(err, "Failed to create service monitor for metrics")
		}
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

func (d *Deployment) createServiceMonitor() error {

	nfsService, err := d.getService()
	if err != nil {
		return err
	}

	// Get a k8s client config
	cfg, err := rest.InClusterConfig()
	if err != nil {
		return err
	}

	// Pass the Service(s) to the helper function, which in turn returns the array of `ServiceMonitor` objects.
	_, err = metrics.CreateServiceMonitors(cfg, d.nfsServer.Namespace, []*corev1.Service{nfsService})
	if err != nil {
		return err
	}

	return nil
}
