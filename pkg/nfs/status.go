package nfs

import (
	"context"
	"fmt"
	"reflect"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)

func (s *Deployment) updateStatus(status *storageosv1.NFSServerStatus) error {

	if reflect.DeepEqual(s.nfsServer.Status, *status) {
		return nil
	}

	// When there's a difference in remote target, broadcast the status change
	// event.
	if s.nfsServer.Status.RemoteTarget != status.RemoteTarget {
		if status.RemoteTarget != "" {
			if s.recorder != nil {
				s.recorder.Event(s.nfsServer, corev1.EventTypeNormal, "ChangedStatus", fmt.Sprintf("NFS server is now functional: %s", status.RemoteTarget))
			}
		}
	}

	// Update subresource status.
	s.nfsServer.Status = *status
	return s.client.Status().Update(context.Background(), s.nfsServer)
}

// getStatus determines the status of the NFS Server deployment.
func (s *Deployment) getStatus() (*storageosv1.NFSServerStatus, error) {

	status := &storageosv1.NFSServerStatus{
		Phase:        storageosv1.PhaseUnknown,
		RemoteTarget: "",
		AccessModes:  "",
	}

	ss, err := s.k8sResourceManager.StatefulSet(s.nfsServer.Name, s.nfsServer.Namespace, nil).Get()
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty status without any error. Resources haven't been
			// created yet.
			return status, nil
		}
		return status, err
	}

	svc, err := s.k8sResourceManager.Service(s.nfsServer.Name, s.nfsServer.Namespace, nil, nil).Get()
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty status without any error. Resources haven't been
			// created yet.
			return status, nil
		}
		return status, err
	}

	// We got both StatefulSet and Service without error, so upgrade to Pending.
	status.Phase = storageosv1.PhasePending

	// Set access mode.
	if s.nfsServer.Spec.Export.Name == "" {
		status.AccessModes = getAccessMode(DefaultAccessType)
	} else {
		status.AccessModes = getAccessMode(s.nfsServer.Spec.Export.Server.AccessMode)
	}

	// If the service is created, set the cluster address as the endpoint.
	if svc.Spec.ClusterIP != "" {
		status.RemoteTarget = svc.Spec.ClusterIP

		// If the NFS pod is also ready, then we can mark the NFS Server as
		// online.
		if ss.Status.ReadyReplicas > 0 {
			status.Phase = storageosv1.PhaseRunning
		}
	}

	return status, nil
}
