package scheduler

import (
	"context"
	"fmt"
	"net/http"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/runtime/inject"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission/types"
)

// PodSchedulerSetter is responsible for mutating and setting pod scheduler
// name.
type PodSchedulerSetter struct {
	client  client.Client
	decoder types.Decoder
	// Provisioners is a list of storage provisioners to check a pod volume
	// against.
	Provisioners []string
	// SchedulerName is the name of the scheduler to mutate pods with.
	SchedulerName string
}

// Check if the Handler interface is implemented.
var _ admission.Handler = &PodSchedulerSetter{}

// Handle handles an admission request and mutates a pod object in the request.
func (p *PodSchedulerSetter) Handle(ctx context.Context, req types.Request) types.Response {
	// Decode the pod in request to a pod variable.
	pod := &corev1.Pod{}
	if err := p.decoder.Decode(req, pod); err != nil {
		return admission.ErrorResponse(http.StatusBadRequest, err)
	}
	// Create a copy of the pod to mutate.
	copy := pod.DeepCopy()
	if err := p.mutatePodsFn(ctx, copy); err != nil {
		return admission.ErrorResponse(http.StatusInternalServerError, err)
	}
	return admission.PatchResponse(pod, copy)
}

// mutatePodFn mutates a given pod with a configured scheduler name if the pod
// is associated with volumes managed by the configured provisioners.
func (p *PodSchedulerSetter) mutatePodsFn(ctx context.Context, pod *corev1.Pod) error {
	managedVols := []corev1.Volume{}

	// Find all the managed volumes.
	for _, vol := range pod.Spec.Volumes {
		ok, err := p.IsManagedVolume(vol, pod.Namespace)
		if err != nil {
			return fmt.Errorf("failed to determine if the volume is managed: %v", err)
		}
		if ok {
			managedVols = append(managedVols, vol)
		}
	}

	// Set scheduler name only if there are managed volumes.
	if len(managedVols) > 0 {
		// Check if StorageOS scheduler is enabled before setting the scheduler.
		cluster, err := p.getCurrentStorageOSCluster()
		if err != nil {
			return err
		}
		if !cluster.Spec.DisableScheduler {
			pod.Spec.SchedulerName = p.SchedulerName
		}
	}

	return nil
}

// Check if the Client interface is implemented.
var _ inject.Client = &PodSchedulerSetter{}

// InjectClient injects a client into object.
func (p *PodSchedulerSetter) InjectClient(c client.Client) error {
	p.client = c
	return nil
}

// Check if the Decoder interface is implemented.
var _ inject.Decoder = &PodSchedulerSetter{}

// InjectDecoder injects a decoder into the object.
func (p *PodSchedulerSetter) InjectDecoder(d types.Decoder) error {
	p.decoder = d
	return nil
}
