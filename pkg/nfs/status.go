package nfs

import (
	"context"
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
)

const (
	// nfsServerPodLabelSelector is the label that can be used to select all the
	// pods of NFS Server.
	nfsServerPodLabelSelector = "nfsserver"
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

	// Check if the StatefulSet exists.
	_, err := s.k8sResourceManager.StatefulSet(s.nfsServer.Name, s.nfsServer.Namespace, nil, nil).Get()
	if err != nil {
		if errors.IsNotFound(err) {
			// Return empty status without any error. Resources haven't been
			// created yet.
			return status, nil
		}
		return status, err
	}

	// Check if the Service exists.
	svc, err := s.k8sResourceManager.Service(s.nfsServer.Name, s.nfsServer.Namespace, nil, nil, nil).Get()
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

		// Get the NFS Server pods and check their status.
		listOpts := []client.ListOption{
			client.MatchingLabels{nfsServerPodLabelSelector: s.nfsServer.Name},
		}
		podList := &corev1.PodList{}
		if err := s.client.List(context.Background(), podList, listOpts...); err != nil {
			return status, err
		}

		// If any of the NFS pods are ready, then we can mark the NFS Server as
		// online.
		for _, pod := range podList.Items {
			if pod.Status.Phase == corev1.PodRunning {
				status.Phase = storageosv1.PhaseRunning
				break
			}
		}
	}

	return status, nil
}
