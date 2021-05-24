// Package resource contains implementation of k8s.Resource interface for
// various k8s resources.
package resource

import (
	"context"
	"errors"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// k8s APIVersion constants.
const (
	APIv1               = "v1"
	APIappsv1           = "apps/v1"
	APIextv1beta1       = "extensions/v1beta1"
	APIrbacv1           = "rbac.authorization.k8s.io/v1"
	APIstoragev1        = "storage.k8s.io/v1"
	APIstoragev1beta1   = "storage.k8s.io/v1beta1"
	APIservicemonitorv1 = "monitoring.coreos.com/v1"
	APIadmissionv1      = "admissionregistration.k8s.io/v1"
)

// Create a k8s resource.  Does not return an error if the resource
// already exists.  It is up to the caller to check beforehand and call Update()
// if required and the object Kind supports it.
func Create(c client.Client, obj runtime.Object) error {
	if err := c.Create(context.Background(), obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// Exists, update must be requested specifically.
			return nil
		}

		kind := obj.GetObjectKind().GroupVersionKind().Kind
		return fmt.Errorf("failed to create %s: %v", kind, err)
	}
	return nil
}

// Update an existing k8s resource.
func Update(c client.Client, obj runtime.Object) error {
	kind := obj.GetObjectKind().GroupVersionKind().Kind

	// Only allow updates on specic kinds.
	if kind != "ConfigMap" {
		return errors.New("update not supported for this object kind")
	}

	if err := c.Update(context.Background(), obj); err != nil {
		return fmt.Errorf("failed to update %s: %v", kind, err)
	}
	return nil
}

// Delete a k8s resource.
func Delete(c client.Client, obj runtime.Object) error {
	if err := c.Delete(context.Background(), obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}
