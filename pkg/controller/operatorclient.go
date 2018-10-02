package controller

import (
	"context"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// OperatorClient is an adapter that implements client.Client interface for operator-SDK.
type OperatorClient struct{}

// Create implements client.Client.
func (oc OperatorClient) Create(ctx context.Context, obj runtime.Object) error {
	return sdk.Create(obj)
}

// Update implements client.Client.
func (oc OperatorClient) Update(ctx context.Context, obj runtime.Object) error {
	return sdk.Update(obj)
}

// Delete implements client.Client.
func (oc OperatorClient) Delete(ctx context.Context, obj runtime.Object) error {
	return sdk.Delete(obj)
}

// Get implements client.Client.
func (oc OperatorClient) Get(ctx context.Context, key types.NamespacedName, obj runtime.Object) error {
	// operator-SDK refers namespace and name from the runtime object. Ignore
	// NamespacedName. sdk.GetOption is not passed at the moment.
	return sdk.Get(obj)
}

// List implements client.Client.
func (oc OperatorClient) List(ctx context.Context, opts *client.ListOptions, obj runtime.Object) error {
	// operator-SDK requires namespace to be passed separately. sdk.ListOption
	// is not passed at the moment.
	return sdk.List(opts.Namespace, obj)
}

// Status implements client.Client.
func (oc OperatorClient) Status() client.StatusWriter {
	return nil
}
