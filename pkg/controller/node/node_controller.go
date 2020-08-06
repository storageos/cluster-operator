package node

import (
	"context"
	"errors"
	"fmt"
	"time"

	storageosapi "github.com/storageos/go-api"
	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	storageosclientcommon "github.com/storageos/cluster-operator/internal/pkg/client/storageos/common"
	storageosclientv1 "github.com/storageos/cluster-operator/internal/pkg/client/storageos/v1"
	storageosclientv2 "github.com/storageos/cluster-operator/internal/pkg/client/storageos/v2"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	storageos "github.com/storageos/cluster-operator/pkg/storageos"
)

var log = logf.Log.WithName("storageos.node")

// Node controller errors.
var (
	ErrCurrentClusterNotFound = errors.New("current cluster not found")
)

const reconcilePeriodSeconds = 5

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

	reconcilePeriod := reconcilePeriodSeconds * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Return this for a immediate retry of the reconciliation loop with the
	// same request object.
	immediateRetryResult := reconcile.Result{Requeue: true}

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
		log.Info("Failed to find current cluster", "error", err)
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
			log.Info("Failed to configure api client", "error", err)
			return reconcileResult, err
		}
	}

	// Check if client is initialized.
	if r.stosClient.client.V1 != nil {
		node, err := r.stosClient.client.GetNodeV1(instance.Name)
		if err != nil {
			if err == storageosapi.ErrNoSuchNode {
				// Not a StorageOS node, skip.
				return reconcile.Result{}, nil
			}
			// Retry immediately if failed to determine if the node is part of the
			// cluster.
			return immediateRetryResult, err
		}

		// Labels can be uninitialized in v1 when empty. Initialize labels
		// before updating with new labels.
		if node.Labels == nil {
			node.Labels = map[string]string{}
		}

		if updateLabels(node.Labels, instance.Labels) {
			if err := r.stosClient.client.UpdateNodeV1(node); err != nil {
				log.Info("Failed to sync node labels", "error", err)
				return reconcileResult, nil
			}
		}
	} else if r.stosClient.client.V2 != nil {
		node, err := r.stosClient.client.GetNodeV2(instance.Name)
		if err != nil {
			if err == storageosclientcommon.ErrUnauthorized {
				r.stosClient = nil
				return immediateRetryResult, nil
			}
			if err == storageosclientcommon.ErrResourceNotFound {
				// Not a StorageOS node, skip.
				return reconcile.Result{}, nil
			}
			// Retry immediately if failed to determine if the node is part of the
			// cluster.
			return immediateRetryResult, err
		}

		if updateLabels(node.Labels, instance.Labels) {
			if err := r.stosClient.client.UpdateNodeV2(node); err != nil {
				if err == storageosclientcommon.ErrUnauthorized {
					r.stosClient = nil
					return immediateRetryResult, nil
				}
				log.Info("Failed to sync node labels", "error", err)
				return reconcileResult, nil
			}
		}
	} else {
		// Retry with an initialized client.
		log.Info("StorageOS API client not initialized")
		return reconcileResult, err
	}

	return reconcile.Result{}, nil
}

// updateLabels takes StorageOS node labels and k8s node labels and updates the
// StorageOS node labels. If there's a change in the labels, it returns a bool
// true.
func updateLabels(stosNodeLabels, k8sNodeLabels map[string]string) bool {
	// Initialize if nil to avoid panic when updating the elements.
	if stosNodeLabels == nil {
		stosNodeLabels = map[string]string{}
	}

	changed := false

	for kKey, kVal := range k8sNodeLabels {
		// Check if k8s label exists in storageos labels.
		if sVal, exists := stosNodeLabels[kKey]; exists {
			// If the label values don't match, update the storageos label
			// value.
			if sVal != kVal {
				stosNodeLabels[kKey] = kVal
				changed = true
			}
		} else {
			// Add new k8s label to storageos labels.
			stosNodeLabels[kKey] = kVal
			changed = true
		}
	}

	return changed
}

// findCurrentCluster finds the running cluster.
func (r *ReconcileNode) findCurrentCluster() (*storageosv1.StorageOSCluster, error) {
	clusterList := &storageosv1.StorageOSClusterList{}
	listOpts := []client.ListOption{}
	if err := r.client.List(context.TODO(), clusterList, listOpts...); err != nil {
		return nil, fmt.Errorf("failed to list clusters: %v", err)
	}

	var currentCluster *storageosv1.StorageOSCluster
	for _, cluster := range clusterList.Items {
		cluster := cluster
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

	// Set the client and the current cluster attributes.
	r.stosClient = &StorageOSClient{
		clusterName:       cluster.GetName(),
		clusterGeneration: cluster.GetGeneration(),
		clusterUID:        cluster.GetUID(),
	}

	// Initialize StorageOS client based on the version of StorageOS cluster.
	if storageos.NodeV2Image(cluster.Spec.GetNodeContainerImage()) {
		r.stosClient.client.Ctx, r.stosClient.client.V2, err = storageosclientv2.NewClientFromSecret(serviceInstance.Spec.ClusterIP, secretInstance)
		if err != nil {
			return fmt.Errorf("failed to create StorageOS v2 client: %v", err)
		}
	} else {
		r.stosClient.client.V1, err = storageosclientv1.NewClientFromSecret(serviceInstance.Spec.ClusterIP, secretInstance)
		if err != nil {
			return fmt.Errorf("failed to create StorageOS v1 client: %v", err)
		}
	}

	return nil
}
