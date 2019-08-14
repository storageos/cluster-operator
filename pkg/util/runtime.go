package util

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// CreateOrUpdateObject creates or updates an existing k8s resource.
func CreateOrUpdateObject(c client.Client, obj runtime.Object) error {
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

// DeleteObject deletes a k8s resource.
func DeleteObject(c client.Client, obj runtime.Object) error {
	if err := c.Delete(context.Background(), obj); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}
