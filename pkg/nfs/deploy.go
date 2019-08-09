package nfs

import (
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

const (
	appName         = "storageos"
	statefulsetKind = "statefulset"

	DefaultNFSPort     = 2049
	DefaultRPCPort     = 111
	DefaultMetricsPort = 9587
)

var log = logf.Log.WithName("storageos.nfsserver")

// Deploy deploys a NFS server.
func (d *Deployment) Deploy() error {
	requestedCapacity := d.nfsServer.Spec.GetRequestedCapacity()
	size := &requestedCapacity

	err := d.ensureService(DefaultNFSPort, DefaultRPCPort, DefaultMetricsPort)
	if err != nil {
		return err
	}
	if err := d.createNFSConfigMap(); err != nil {
		return err
	}
	if err := d.createStatefulSet(size, DefaultNFSPort, DefaultRPCPort, DefaultMetricsPort); err != nil {
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
func labelsForStatefulSet(name string, labels map[string]string) map[string]string {
	if labels == nil {
		labels = make(map[string]string)
	}

	labels["app"] = appName
	labels["nfsserver"] = name

	// TODO: setting fenced should only be done if we _know_ that fencing hasn't
	// been disabled else provisioning will fail
	labels["storageos.com/fenced"] = "true"
	return labels
}
