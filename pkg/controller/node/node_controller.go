package node

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	storageosapi "github.com/storageos/go-api"
	storageostypes "github.com/storageos/go-api/types"
)

var log = logf.Log.WithName("controller_node")

// Add creates a new Node Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileNode{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("node-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Nodes
	// return c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{})

}

var _ reconcile.Reconciler = &ReconcileNode{}

// ReconcileNode reconciles a Node object
type ReconcileNode struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	api    *storageosapi.Client
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNode) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Node")

	// Fetch the Node instance
	instance := &corev1.Node{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	log.Info("instance: %#v", instance)

	// Get a StorageOS api client.
	if r.api == nil {
		client, err := r.apiClient()
		if err != nil {
			return reconcile.Result{}, err
		}
		r.api = client
	}

	// Sync labels to StorageOS node object.
	if err = r.syncLabels(instance.Name, instance.Labels); err != nil {
		return reconcile.Result{}, err
	}
	return reconcile.Result{}, nil
}

// SyncNodeLabels applies the Kubernetes node labels to StorageOS node objects.
func (r *ReconcileNode) syncLabels(name string, labels map[string]string) error {

	if len(name) == 0 || len(labels) == 0 {
		return nil
	}

	// Get StorageOS node
	node, err := r.api.Node(name)
	if err != nil {
		return err
	}

	log.Info("node: %#v", node)

	original := node.Labels

	if len(node.Labels) == 0 {
		node.Labels = make(map[string]string)
	}

	// Add/replace each Kubernetes node label
	for k, v := range labels {
		node.Labels[k] = v
	}

	// No updates, return
	if len(node.Labels) == len(original) {
		changed := false
		for k, v := range original {
			if n, ok := node.Labels[k]; !ok || n != v {
				changed = true
				break
			}
		}
		if !changed {
			return nil
		}
	}

	ctx, cancel := context.WithTimeout(context.TODO(), 2*time.Second)
	defer cancel()

	// Update StorageOS node
	opts := storageostypes.NodeUpdateOptions{
		ID:          node.ID,
		Name:        node.Name,
		Description: node.Description,
		Labels:      node.Labels,
		Cordon:      node.Cordon,
		Drain:       node.Drain,
		Context:     ctx,
	}

	_, err = r.api.NodeUpdate(opts)
	return err
}

// apiClient returns a StorageOS api client for the current cluster.
func (r *ReconcileNode) apiClient() (*storageosapi.Client, error) {
	cluster, err := r.findCurrentCluster()
	if err != nil {
		return nil, err
	}
	return r.apiClientForCluster(cluster)
}

// findCurrentCluster finds the running cluster.
func (r *ReconcileNode) findCurrentCluster() (*storageosv1.StorageOSCluster, error) {
	clusterList := &storageosv1.StorageOSClusterList{}
	if err := r.client.List(context.TODO(), &client.ListOptions{}, clusterList); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %v", err)
	}

	var currentCluster *storageosv1.StorageOSCluster
	for _, cluster := range clusterList.Items {
		// The cluster with Phase "Running" is the only active cluster.
		if cluster.Status.Phase == storageosv1.ClusterPhaseRunning {
			currentCluster = &cluster
			break
		}
	}

	if currentCluster == nil {
		return nil, fmt.Errorf("failed to find currently running cluster")
	}

	return currentCluster, nil
}

// apiClientForCluster returns an api client for the StorageOS cluster.
func (r *ReconcileNode) apiClientForCluster(cluster *storageosv1.StorageOSCluster) (*storageosapi.Client, error) {

	serviceNamespacedName := types.NamespacedName{
		Namespace: cluster.Spec.GetResourceNS(),
		Name:      cluster.Spec.GetServiceName(),
	}
	serviceInstance := &corev1.Service{}
	err := r.client.Get(context.TODO(), serviceNamespacedName, serviceInstance)
	if err != nil {
		return nil, err
	}

	client, err := storageosapi.NewVersionedClient(strings.Join([]string{serviceInstance.Spec.ClusterIP, storageosapi.DefaultPort}, ":"), storageosapi.DefaultVersionStr)
	if err != nil {
		return nil, err
	}

	secretNamespacedName := types.NamespacedName{
		Namespace: cluster.Spec.SecretRefNamespace,
		Name:      cluster.Spec.SecretRefName,
	}
	secretInstance := &corev1.Secret{}
	err = r.client.Get(context.TODO(), secretNamespacedName, secretInstance)
	if err != nil {
		return nil, err
	}

	client.SetUserAgent("cluster-operator")
	client.SetAuth(string(secretInstance.Data["apiUsername"]), string(secretInstance.Data["apiPassword"]))

	return client, nil
}
