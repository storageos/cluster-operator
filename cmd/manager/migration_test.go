package main

import (
	"context"
	"testing"

	admissionv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestWebhookMigration(t *testing.T) {
	testNS := "test-ns"

	// Mutating webhook configuration.
	whc := &admissionv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name: oldWebhookResourceName,
		},
	}

	// Webhook Service.
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      oldWebhookResourceName,
			Namespace: testNS,
		},
	}

	testcases := []struct {
		name              string
		existingResources []runtime.Object
	}{
		{
			name: "webhook resources don't exist",
		},
		{
			name:              "webhook resources exist",
			existingResources: []runtime.Object{whc, svc},
		},
	}

	for _, tc := range testcases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			cli := fake.NewFakeClient(tc.existingResources...)

			ctx := context.TODO()

			if err := webhookMigration(ctx, cli, log, testNS); err != nil {
				t.Errorf("failed running webhook migration: %v", err)
			}

			// Check if the resources exist.
			checkObjectExists(t, ctx, cli, whc)
			checkObjectExists(t, ctx, cli, svc)
		})
	}
}

func checkObjectExists(t *testing.T, ctx context.Context, cli client.Client, obj runtime.Object) {
	key, keyErr := client.ObjectKeyFromObject(obj)
	if keyErr != nil {
		t.Errorf("failed to get object key from object: %v", keyErr)
	}
	if err := cli.Get(ctx, key, obj); err == nil {
		t.Error("GET after DELETE should fail")
	}
}
