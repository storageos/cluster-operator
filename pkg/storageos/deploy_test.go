package storageos

import (
	"context"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	api "github.com/storageos/storageoscluster-operator/pkg/apis/cluster/v1alpha1"
)

func TestCreateNamespace(t *testing.T) {
	const (
		apiVersion = "fooVersion"
		kind       = "StorageOSCluster"
	)

	c := fake.NewFakeClient()

	stosCluster := &api.StorageOSCluster{
		TypeMeta: metav1.TypeMeta{
			APIVersion: apiVersion,
			Kind:       kind,
		},
	}

	deploy := Deployment{
		client: c,
		stos:   stosCluster,
	}

	if err := deploy.createNamespace(); err != nil {
		t.Error("failed:", err)
	}

	// Fetch the created namespace and check if it's a child of StorageOSCluster.
	nsName := types.NamespacedName{Name: "storageos"}
	wantNS := &v1.Namespace{}
	if err := c.Get(context.TODO(), nsName, wantNS); err != nil {
		t.Error("failed to get the created object:", err)
	}

	owner := wantNS.GetOwnerReferences()[0]
	if owner.APIVersion != apiVersion {
		t.Errorf("unexpected object owner api version:\n\t(WNT) %s\n\t(GOT) %s", apiVersion, owner.APIVersion)
	}
	if owner.Kind != kind {
		t.Errorf("unexpected object owner kindL\n\t(WNT) %s\n\t(GOT) %s", kind, owner.Kind)
	}
}
