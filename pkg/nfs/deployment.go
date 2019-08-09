package nfs

import (
	"context"
	"fmt"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Deployment struct {
	client    client.Client
	nfsServer *storageosv1.NFSServer
	recorder  record.EventRecorder
	scheme    *runtime.Scheme
}

func NewDeployment(client client.Client, nfsServer *storageosv1.NFSServer, recorder record.EventRecorder, scheme *runtime.Scheme) *Deployment {
	return &Deployment{
		client:    client,
		nfsServer: nfsServer,
		recorder:  recorder,
		scheme:    scheme,
	}
}

// createOrUpdateObject attempts to create a given object. If the object already
// exists the existing object is updated.
func (d *Deployment) createOrUpdateObject(obj runtime.Object) error {
	if err := d.client.Create(context.Background(), obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return d.client.Update(context.Background(), obj)
		}

		kind := obj.GetObjectKind().GroupVersionKind().Kind
		return fmt.Errorf("failed to create %s: %v", kind, err)
	}
	return nil
}

// deleteObject deletes a given runtime object.
func (d *Deployment) deleteObject(obj runtime.Object) error {
	if err := d.client.Delete(context.Background(), obj); err != nil {
		// If not found, the object has already been deleted.
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}
