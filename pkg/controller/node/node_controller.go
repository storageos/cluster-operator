package node

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	storageosapi "github.com/storageos/go-api"
	storageostypes "github.com/storageos/go-api/types"
)

var log = ctrl.Log.WithName("node")

// Node controller errors.
var (
	ErrCurrentClusterNotFound = errors.New("current cluster not found")
	ErrNoAPIClient            = errors.New("api client not available")
)

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

	// Watch for changes to Nodes.
	return c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForObject{})
}

var _ reconcile.Reconciler = &ReconcileNode{}

// ReconcileNode reconciles a Node object
type ReconcileNode struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client     client.Client
	scheme     *runtime.Scheme
	stosClient *StorageOSClient
}

// Reconcile reads that state of the cluster for a Node object and makes changes based on the state read
// and what is in the Node.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNode) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	log := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	// log.Info("Reconciling Node")

	reconcilePeriod := 5 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the Node instance
	instance := &corev1.Node{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if kerrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult, err
	}

	// Get the current storageos cluster.
	cluster, err := r.findCurrentCluster()
	if err != nil {
		if err == ErrCurrentClusterNotFound {
			// Do not requeue request. No StorageOS cluster is installed.
			return reconcile.Result{}, nil
		}
		// Requeue the request in order to retry getting the cluster.
		log.Error(err, "failed to find current cluster")
		return reconcileResult, err
	}

	// Compare the cluster names, generations and UUIDs to check if it's
	// the same cluster. Update the client if client cluster name,
	// generation or UID are different from current cluster.
	if r.stosClient == nil ||
		r.stosClient.clusterName != cluster.GetName() ||
		r.stosClient.clusterGeneration != cluster.GetGeneration() ||
		r.stosClient.clusterUID != cluster.GetUID() {

		if err := r.setClientForCluster(cluster); err != nil {
			log.Error(err, "failed to configure api client")
			return reconcileResult, err
		}
	}

	// Sync labels to StorageOS node object.
	if err = r.syncLabels(instance.Name, instance.Labels); err != nil {
		log.Error(err, "failed to sync labels, api may not be ready")
		// Error syncing labels - requeue the request.
		return reconcileResult, err
	}

	return reconcileResult, nil
}

// SyncNodeLabels applies the Kubernetes node labels to StorageOS node objects.
func (r *ReconcileNode) syncLabels(name string, labels map[string]string) error {
	if len(name) == 0 || len(labels) == 0 {
		return nil
	}

	if r.stosClient == nil {
		return ErrNoAPIClient
	}

	// Get StorageOS node
	node, err := r.stosClient.Node(name)
	if err != nil {
		return err
	}

	// Initialize the map if empty.
	if len(node.Labels) == 0 {
		node.Labels = make(map[string]string)
	}

	// Check if the k8s node labels already exist in storageos node labels.
	// If there's no update, or addition, do not update the storageos node.
	changed := false
	for k, v := range labels {
		// If the label already exists, compare the new and old values.
		if v2, ok := node.Labels[k]; ok {
			if v2 != v {
				changed = true
				// Set the new value.
				node.Labels[k] = v2
			}
		} else {
			// Add the new label.
			node.Labels[k] = v
			changed = true
		}
	}

	// Return if there's no update or addition.
	if !changed {
		return nil
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

	_, err = r.stosClient.NodeUpdate(opts)
	return err
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
		return nil, ErrCurrentClusterNotFound
	}

	return currentCluster, nil
}

// setClientForCluster sets an api client for the given StorageOS cluster.
func (r *ReconcileNode) setClientForCluster(cluster *storageosv1.StorageOSCluster) error {
	// Get the storageos service resource to obtain the service IP from it.
	// This service IP is used to create the storageos API client.
	serviceNamespacedName := types.NamespacedName{
		Namespace: cluster.Spec.GetResourceNS(),
		Name:      cluster.Spec.GetServiceName(),
	}
	serviceInstance := &corev1.Service{}
	err := r.client.Get(context.TODO(), serviceNamespacedName, serviceInstance)
	if err != nil {
		return err
	}

	// Create a versioned storageos client.
	client, err := storageosapi.NewVersionedClient(strings.Join([]string{serviceInstance.Spec.ClusterIP, storageosapi.DefaultPort}, ":"), storageosapi.DefaultVersionStr)
	if err != nil {
		return err
	}

	// Obtain the storageos API secrets to be used in the client.
	secretNamespacedName := types.NamespacedName{
		Namespace: cluster.Spec.SecretRefNamespace,
		Name:      cluster.Spec.SecretRefName,
	}
	secretInstance := &corev1.Secret{}
	err = r.client.Get(context.TODO(), secretNamespacedName, secretInstance)
	if err != nil {
		return err
	}

	client.SetUserAgent("cluster-operator")
	client.SetAuth(string(secretInstance.Data["apiUsername"]), string(secretInstance.Data["apiPassword"]))

	// Set the client and the current cluster attributes.
	r.stosClient = &StorageOSClient{
		Client:            client,
		clusterName:       cluster.GetName(),
		clusterGeneration: cluster.GetGeneration(),
		clusterUID:        cluster.GetUID(),
	}

	return nil
}
