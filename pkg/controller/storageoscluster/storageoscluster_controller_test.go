package storageoscluster

import (
	"context"
	"testing"

	storageosapis "github.com/storageos/cluster-operator/pkg/apis"
	stosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	corev1 "k8s.io/api/core/v1"
	apiextensionsv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clientgofake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var gvk = schema.GroupVersionKind{
	Group:   "storageos.com",
	Version: "v1",
	Kind:    "StorageOSCluster",
}

func TestGenerateJoinToken(t *testing.T) {
	testcases := []struct {
		name      string
		nodes     []*corev1.Node
		stosSpec  stosv1.StorageOSClusterSpec
		wantToken string
		wantError bool
	}{
		{
			name: "three-node no selector",
			nodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "fake-node1"},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "0.0.0.0"}},
					},
				},
				{ObjectMeta: metav1.ObjectMeta{Name: "fake-node2"},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "0.0.0.1"}},
					},
				},
				{ObjectMeta: metav1.ObjectMeta{Name: "fake-node3"},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "0.0.0.2"}},
					},
				},
			},
			stosSpec:  stosv1.StorageOSClusterSpec{},
			wantToken: "0.0.0.0,0.0.0.1,0.0.0.2",
			wantError: false,
		},
		{
			name: "three-node with node selector",
			nodes: []*corev1.Node{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "fake-node1",
						Labels: map[string]string{"foo": "baz"},
					},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "0.0.0.0"}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "fake-node2",
						Labels: map[string]string{"foo4": "baz4"},
					},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "0.0.0.1"}},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "fake-node3",
						Labels: map[string]string{"foo0": "baz0", "foo": "baz"},
					},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "0.0.0.2"}},
					},
				},
			},
			stosSpec: stosv1.StorageOSClusterSpec{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{{
					MatchExpressions: []corev1.NodeSelectorRequirement{{
						Key:      "foo",
						Operator: corev1.NodeSelectorOpIn,
						Values:   []string{"baz"},
					}},
				}},
			},
			wantToken: "0.0.0.0,0.0.0.2",
			wantError: false,
		},
		{
			name:      "no nodes",
			wantToken: "",
			wantError: false,
		},
		{
			name: "unsupported node selector operator",
			// Need at least one node to run the node selector operator block.
			nodes: []*corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "fake-node1"},
					Status: corev1.NodeStatus{
						Addresses: []corev1.NodeAddress{{Type: corev1.NodeInternalIP, Address: "0.0.0.0"}},
					},
				},
			},
			stosSpec: stosv1.StorageOSClusterSpec{
				NodeSelectorTerms: []corev1.NodeSelectorTerm{{
					MatchExpressions: []corev1.NodeSelectorRequirement{{
						Key:      "foo",
						Operator: corev1.NodeSelectorOpDoesNotExist,
						Values:   []string{"baz"},
					}},
				}},
			},
			wantError: true,
		},
	}

	for _, tc := range testcases {
		// Create controller fake client.
		controllerClient := fake.NewFakeClient()

		// Create fake node objects.
		for _, node := range tc.nodes {
			if err := controllerClient.Create(context.Background(), node); err != nil {
				t.Fatal(err)
			}
		}
		testScheme := runtime.NewScheme()

		// Register all the schemes.
		if err := kscheme.AddToScheme(testScheme); err != nil {
			t.Fatal(err)
		}
		if err := apiextensionsv1beta1.AddToScheme(testScheme); err != nil {
			t.Fatal(err)
		}
		if err := storageosapis.AddToScheme(testScheme); err != nil {
			t.Fatal(err)
		}

		// Create kubernetes client for event recorder.
		kubeClient := clientgofake.NewSimpleClientset()

		// Create event broadcaster to be used in reconcile object.
		eventBroadcaster := record.NewBroadcaster()
		eventBroadcaster.StartRecordingToSink(
			&typedcorev1.EventSinkImpl{
				Interface: kubeClient.CoreV1().Events(""),
			},
		)
		recorder := eventBroadcaster.NewRecorder(
			scheme.Scheme,
			corev1.EventSource{Component: "storageoscluster-operator"},
		)

		// Current cluster.
		stosCluster := &stosv1.StorageOSCluster{
			TypeMeta: metav1.TypeMeta{
				APIVersion: gvk.GroupVersion().String(),
				Kind:       gvk.Kind,
			},
			Spec: tc.stosSpec,
		}

		r := ReconcileStorageOSCluster{
			client:         controllerClient,
			scheme:         testScheme,
			k8sVersion:     "foo-version",
			recorder:       recorder,
			currentCluster: NewStorageOSCluster(stosCluster),
		}

		token, err := r.generateJoinToken(stosCluster)
		if !tc.wantError && err != nil {
			t.Fatalf("expected no error but got one: %v", err)
		}
		if tc.wantError && err == nil {
			t.Error("expected error but got none")
		}

		if token != tc.wantToken {
			t.Errorf("unexpected join token:\n\t(GOT) %v\n\t(WNT) %v", token, tc.wantToken)
		}
	}
}
