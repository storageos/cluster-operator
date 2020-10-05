// Package resource contains implementation of k8s.Resource interface for
// various k8s resources.
package resource

import (
	"context"
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
)

// CreateOrUpdate creates or updates an existing k8s resource.
func CreateOrUpdate(c client.Client, obj runtime.Object) error {
	if err := c.Create(context.Background(), obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			// TODO: Support update.
			// Check for update option and update the object.
			// return c.Update(context.Background(), obj)

			// Exists, no update.
			return nil
		}

		kind := obj.GetObjectKind().GroupVersionKind().Kind
		return fmt.Errorf("failed to create %s: %v", kind, err)
	}
	return nil
}

// Delete deletes a k8s resource.
func Delete(c client.Client, obj runtime.Object) error {
	if err := c.Delete(context.Background(), obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}
