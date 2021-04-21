package main

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	oputils "github.com/operator-framework/operator-sdk/pkg/k8sutil"
	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/storageos/cluster-operator/pkg/util/k8s/resource"
)

// **Upgrade Migrations**
//
// Webhook configuration and service migration:
//
// Before cluster-operator v2.4.0, the admission webhook server ran in
// the cluster-operator itself. Some old controller-runtime admission
// controller tooling created the webhook configurations and services at
// operator startup. These resources are unmanaged and are left behind even
// after uninstalling the cluster-operator. In cluster-operator v2.4.0, the
// admission webhook server is moved to the api-manager component and the
// new webhook configurations and services are managed by the storageos
// cluster controller. The old resources can't be reused because the new
// resources have different names and configurations. They have to be
// deleted. This migration ensures that the known old resources are cleaned
// up before starting the controllers.
//
// Known resources:
//   - storageos-scheduler-webhook MutatingWebhookConfiguration, cluster
// scoped.
//   - storageos-scheduler-webhook Service, same namespace as the
// cluster-operator.

const (
	// oldWebhookResourceName is the name of the old webhook resources.
	oldWebhookResourceName = "storageos-scheduler-webhook"

	// migrationTimeoutDuration is the duration for which the migration is
	// allowed to run.
	migrationTimeoutDuration = 20 * time.Second
)

// migrationFailLog logs the migration failure info.
func migrationFailLog(log logr.Logger, kind, key, err string) {
	log.Info("failed to delete resource",
		"kind", kind,
		"key", key,
		"error", err,
	)
}

// migrate runs upgrade migrations.
func migrate(c client.Client, log logr.Logger) error {
	log = log.WithName("migration")

	ns, err := oputils.GetOperatorNamespace()
	if err != nil {
		return err
	}

	// Use a context with timeout to be able to exit the migration within a
	// reasonable time if the operations get stuck.
	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeoutDuration)
	defer cancel()

	// Run migration actions.

	if err := webhookMigration(ctx, c, log, ns); err != nil {
		return err
	}

	log.Info("migration successful")

	return nil
}

// webhookMigration runs the webhook migration, deleting the old webhook
// related resources.
func webhookMigration(ctx context.Context, c client.Client, log logr.Logger, namespace string) error {
	// Check if the known resources exist and delete them if found, no-op if
	// not found.
	mutatingWHC := &admissionv1.MutatingWebhookConfiguration{}
	mhcKey := types.NamespacedName{Name: oldWebhookResourceName}
	if err := deleteIfFound(ctx, c, mhcKey, mutatingWHC); err != nil {
		migrationFailLog(log, resource.MutatingWebhookConfigurationKind, mhcKey.String(), err.Error())
		return err
	}

	webhookSvc := &corev1.Service{}
	whsKey := types.NamespacedName{
		Name:      oldWebhookResourceName,
		Namespace: namespace,
	}
	if err := deleteIfFound(ctx, c, whsKey, webhookSvc); err != nil {
		migrationFailLog(log, resource.ServiceKind, whsKey.String(), err.Error())
		return err
	}

	return nil
}

// deleteIfFound deletes the given object if it exists.
func deleteIfFound(ctx context.Context, c client.Client, key client.ObjectKey, obj runtime.Object) error {
	if getErr := c.Get(ctx, key, obj); getErr != nil {
		// If the object is not found, return nil, nothing to do. Else, return
		// the received error.
		if errors.IsNotFound(getErr) {
			return nil
		}
		return getErr
	}

	// Delete the found object.
	if delErr := c.Delete(ctx, obj); delErr != nil {
		// Ignore not found error.
		if errors.IsNotFound(delErr) {
			return nil
		}
		return delErr
	}

	return nil
}
