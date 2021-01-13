package storageoscluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/storageos/cluster-operator/internal/pkg/storageoscluster"
	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/pkg/util/k8sutil"
)

var log = logf.Log.WithName("storageos.cluster")

const (
	clusterFinalizer = "finalizer.storageoscluster.storageos.com"

	reconcilePeriodSeconds = 15
)

// Add creates a new StorageOSCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	// Get k8s version from client and set the version in ReconcileStorageOSCluster.
	clientset := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	k := k8sutil.NewK8SOps(clientset, log)
	version, err := k.GetK8SVersion()
	if err != nil {
		return err
	}

	log.WithValues("k8s", version).Info("Adding cluster controller")

	return add(mgr, newReconciler(mgr, version))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sVersion string) reconcile.Reconciler {
	return &ReconcileStorageOSCluster{
		client:          mgr.GetClient(),
		scheme:          mgr.GetScheme(),
		k8sVersion:      k8sVersion,
		recorder:        mgr.GetEventRecorderFor("storageoscluster-operator"),
		discoveryClient: discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig()),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("storageoscluster-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource StorageOSCluster
	err = c.Watch(&source.Kind{Type: &storageosv1.StorageOSCluster{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileStorageOSCluster{}

// ReconcileStorageOSCluster reconciles a StorageOSCluster object
type ReconcileStorageOSCluster struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client          client.Client
	scheme          *runtime.Scheme
	k8sVersion      string
	recorder        record.EventRecorder
	currentCluster  *StorageOSCluster
	discoveryClient discovery.DiscoveryInterface
}

// UpdateCurrentCluster checks if there are any existing cluster and updates the
// current cluster with the new cluster is no existing cluster is found.
func (r *ReconcileStorageOSCluster) UpdateCurrentCluster(cluster *storageosv1.StorageOSCluster) error {
	cc, err := storageoscluster.GetCurrentStorageOSCluster(r.client)
	if err != nil {
		if err == storageoscluster.ErrNoCluster {
			// If there's no existing cluster, set the passed cluster as the
			// current cluster.
			r.SetCurrentCluster(cluster)
		} else {
			return fmt.Errorf("failed to get current cluster: %v", err)
		}
	} else {
		r.SetCurrentCluster(cc)
	}
	return nil
}

// SetCurrentCluster sets the currently active cluster in the controller.
func (r *ReconcileStorageOSCluster) SetCurrentCluster(cluster *storageosv1.StorageOSCluster) {
	r.currentCluster = NewStorageOSCluster(cluster)
}

// ResetCurrentCluster resets the current cluster of the controller.
func (r *ReconcileStorageOSCluster) ResetCurrentCluster() {
	r.currentCluster = nil
}

// Reconcile reads that state of the cluster for a StorageOSCluster object and makes changes based on the state read
// and what is in the StorageOSCluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileStorageOSCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	log := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	// log.Info("Reconciling Cluster")

	// Return this for a retry of the reconciliation loop after a period of
	// time.
	reconcilePeriod := reconcilePeriodSeconds * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Return this for a immediate retry of the reconciliation loop with the
	// same request object.
	immediateRetryResult := reconcile.Result{Requeue: true}

	// Fetch the StorageOSCluster instance
	instance := &storageosv1.StorageOSCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return immediateRetryResult, err
	}

	// Set as the current cluster if there's no current cluster.
	if err := r.UpdateCurrentCluster(instance); err != nil {
		log.Info("Failed to update current cluster", "error", err)
		// Failed to determine or set current cluster, requeue the request.
		return immediateRetryResult, nil
	}

	// Check if the cluster instance is marked to be deleted, which is indicated
	// by the deletion timestamp being set.
	if instance.GetDeletionTimestamp() != nil {
		if contains(instance.GetFinalizers(), clusterFinalizer) {
			// Update status subresource.
			instance.Status.Phase = storageosv1.ClusterPhaseTerminating
			if err := r.client.Status().Update(context.TODO(), instance); err != nil {
				log.Info("Failed to update cluster status", "error", err)
				return immediateRetryResult, nil
			}

			// Finalize the cluster.
			if err := r.finalizeCluster(instance); err != nil {
				log.Info("Failed to finalize cluster", "error", err)
				return immediateRetryResult, nil
			}

			// Remove finalizer and update cluster status.
			instance.SetFinalizers(remove(instance.GetFinalizers(), clusterFinalizer))
			if err := r.client.Update(context.TODO(), instance); err != nil {
				log.Info("Failed to update cluster finalizers", "error", err)
				return immediateRetryResult, nil
			}
		}
		// Return and do not requeue. Successful deletion.
		return reconcile.Result{}, nil
	}

	// Add finalizer if not exists already.
	if !contains(instance.GetFinalizers(), clusterFinalizer) {
		if err := r.addFinalizer(instance); err != nil {
			log.Info("Failed to update cluster with finalizer", "error", err)
			// Requeue if adding finalizer fails for a retry.
			return immediateRetryResult, nil
		}
	}

	// If the event doesn't belongs to the current cluster, do not reconcile.
	// There must be only a single instance of storageos in a cluster.
	if !r.currentCluster.IsCurrentCluster(instance) {
		err := fmt.Errorf("can't create more than one storageos cluster")
		r.recorder.Event(instance, corev1.EventTypeWarning, "FailedCreation", err.Error())

		// Set the cluster status to pending.
		instance.Status.Phase = storageosv1.ClusterPhasePending
		if err := r.client.Status().Update(context.Background(), instance); err != nil {
			log.Info("Failed to update cluster status", "error", err)
			// Requeue so that a status update is attempted again.
			return immediateRetryResult, nil
		}

		// Requeue the request so that this cluster is deployed as soon as it
		// becomes the current cluster.
		return immediateRetryResult, nil
	} else if r.currentCluster.cluster.GetUID() != instance.GetUID() {
		// If the cluster name and namespace match with the current cluster, but
		// the resource UIDs are different, maybe the current cluster reset
		// failed when the previous cluster was deleted. The same cluster was
		// created again and has a different UID. Create and assign a new
		// current cluster.
		log.WithValues("current", r.currentCluster.cluster.GetUID(), "new", instance.GetUID()).Info("Replacing cluster id")
		r.SetCurrentCluster(instance)
	}

	if err := r.reconcile(instance); err != nil {
		log.Info("Reconcile failed", "error", err)
		return immediateRetryResult, nil
	}

	// Requeue to reconcile after a period of time.
	return reconcileResult, nil
}

// finalizeCluster performs cleanup of the resources before deleting the cluster
// custom resource. Cluster deployment is deleted only when passed cluster is
// the currently running StorageOS cluster. No cleanup is needed for a pending
// cluster.
func (r *ReconcileStorageOSCluster) finalizeCluster(m *storageosv1.StorageOSCluster) error {
	// Check if the cluster being finalized is the currently running cluster.
	if r.currentCluster.cluster.Name == m.Name {
		r.recorder.Event(m, corev1.EventTypeNormal, "Terminating", "Deleting all the resources...")
		if err := r.currentCluster.DeleteDeployment(r); err != nil {
			return fmt.Errorf("failed to delete the cluster: %v", err)
		}
	}

	return nil
}

// addFinalizer adds a finalizer on the cluster object to avoid instant deletion
// of the object without finalizing it.
func (r *ReconcileStorageOSCluster) addFinalizer(m *storageosv1.StorageOSCluster) error {
	log.Info("Adding Finalizer for the StorageOSCluster")
	m.SetFinalizers(append(m.GetFinalizers(), clusterFinalizer))

	// Update CR.
	if err := r.client.Update(context.TODO(), m); err != nil {
		return err
	}
	return nil
}

func (r *ReconcileStorageOSCluster) reconcile(m *storageosv1.StorageOSCluster) error {
	if m.Spec.Pause {
		// Do not reconcile, the operator is paused for the cluster.
		return nil
	}

	// Update the spec values. This ensures that the default values are applied
	// when fields are not set in the spec.
	updated, err := r.updateSpec(m)
	if err != nil {
		return err
	}

	// If updated, update the current cluster and return, as the current
	// instance is outdated.
	if updated {
		r.SetCurrentCluster(m)
		return nil
	}

	if err := r.currentCluster.Deploy(r); err != nil {
		// Ignore "Operation cannot be fulfilled" error. It happens when the
		// actual state of object is different from what is known to the operator.
		// Operator would resync and retry the failed operation on its own.
		if !strings.HasPrefix(err.Error(), "Operation cannot be fulfilled") {
			r.recorder.Event(m, corev1.EventTypeWarning, "FailedCreation", err.Error())
		}

		// Set the status to pending.
		r.currentCluster.cluster.Status.Phase = storageosv1.ClusterPhasePending
		if err := r.client.Status().Update(context.Background(), r.currentCluster.cluster); err != nil {
			return err
		}

		return err
	}

	return nil
}

// updateSpec takes a StorageOSCluster CR and updates the CR properties with
// defaults and inferred values. It returns true if there was an update. This
// result can be used to decide if the caller should continue with reconcile or
// return from reconcile due to an outdated CR instance.
func (r *ReconcileStorageOSCluster) updateSpec(m *storageosv1.StorageOSCluster) (bool, error) {
	needUpdate := false

	// Check updates for string properties.

	join, err := r.generateJoinToken(m)
	if err != nil {
		return false, err
	}

	properties := map[*string]string{
		&m.Spec.Namespace:                  m.Spec.GetResourceNS(),
		&m.Spec.Images.NodeContainer:       m.Spec.GetNodeContainerImage(),
		&m.Spec.Images.InitContainer:       m.Spec.GetInitContainerImage(),
		&m.Spec.Images.APIManagerContainer: m.Spec.GetAPIManagerImage(),
		&m.Spec.Service.Name:               m.Spec.GetServiceName(),
		&m.Spec.Service.Type:               m.Spec.GetServiceType(),
		&m.Spec.Join:                       join,
	}

	if !m.Spec.DisableScheduler {
		properties[&m.Spec.Images.KubeSchedulerContainer] = m.Spec.GetKubeSchedulerImage(r.k8sVersion)
	}

	// CSI related string properties. These must be set always because CSI is
	// the only supported deployment.
	properties[&m.Spec.Images.CSINodeDriverRegistrarContainer] = m.Spec.GetCSINodeDriverRegistrarImage()
	properties[&m.Spec.Images.CSILivenessProbeContainer] = m.Spec.GetCSILivenessProbeImage()
	properties[&m.Spec.Images.CSIExternalProvisionerContainer] = m.Spec.GetCSIExternalProvisionerImage()
	properties[&m.Spec.Images.CSIExternalAttacherContainer] = m.Spec.GetCSIExternalAttacherImage()

	// Add external resizer image if storageos v2 and supported k8s
	// version.
	if storageos.CSIExternalResizerSupported(r.k8sVersion) {
		properties[&m.Spec.Images.CSIExternalResizerContainer] = m.Spec.GetCSIExternalResizerImage()
	}

	properties[&m.Spec.CSI.DeploymentStrategy] = m.Spec.GetCSIDeploymentStrategy()

	// Ingress related string properties.
	if m.Spec.Ingress.Enable {
		properties[&m.Spec.Ingress.Hostname] = m.Spec.GetIngressHostname()
	}

	for k, v := range properties {
		if updateString(k, v) {
			needUpdate = true
		}
	}

	// Check updates for int properties.

	intProperties := map[*int]int{
		&m.Spec.Service.ExternalPort: m.Spec.GetServiceExternalPort(),
		&m.Spec.Service.InternalPort: m.Spec.GetServiceInternalPort(),
	}

	for k, v := range intProperties {
		if updateInt(k, v) {
			needUpdate = true
		}
	}

	// Check boolean properties.

	// All the CSI options must be enabled. Non-CSI deployment are not
	// supported anymore.
	boolProperties := map[*bool]bool{
		&m.Spec.CSI.Enable:                       true,
		&m.Spec.CSI.EnableControllerPublishCreds: true,
		&m.Spec.CSI.EnableProvisionCreds:         true,
		&m.Spec.CSI.EnableNodePublishCreds:       true,
		&m.Spec.CSI.EnableControllerExpandCreds:  true,
	}

	for k, v := range boolProperties {
		if updateBool(k, v) {
			needUpdate = true
		}
	}

	if needUpdate {
		// Update CR.
		err := r.client.Update(context.TODO(), m)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

// updateString compares the string value of valA with valB and if there's a
// mismatch, assigns valB as the value of valA and returns true. If the values
// are equal, false is returned.
func updateString(valA *string, valB string) bool {
	if *valA != valB {
		*valA = valB
		return true
	}
	return false
}

// updateInt compares the int value of valA with valB and if there's a mismatch,
// assigns valB as the value of valA and returns true. If the values are equal,
// false is returned.
func updateInt(valA *int, valB int) bool {
	if *valA != valB {
		*valA = valB
		return true
	}
	return false
}

// updateBool compares the bool value of valA with valB and if there's a
// mismatch, assigns valB as the value of valA and returns true. If the values
// are equal, false is returned.
func updateBool(valA *bool, valB bool) bool {
	if *valA != valB {
		*valA = valB
		return true
	}
	return false
}

// generateJoinToken performs node selection based on NodeSelectorTerms if
// specified, and forms a join token by combining the node IPs.
func (r *ReconcileStorageOSCluster) generateJoinToken(m *storageosv1.StorageOSCluster) (string, error) {
	// Get a new list of all the nodes.
	nodeList := storageos.NodeList()
	listOpts := []client.ListOption{}
	if err := r.client.List(context.Background(), nodeList, listOpts...); err != nil {
		return "", fmt.Errorf("failed to list nodes: %v", err)
	}

	toleratedNodes := []corev1.Node{}
	for _, node := range nodeList.Items {
		// Skip nodes which have taints not tolerated by storageos.
		ok, _ := getMatchingTolerations(node.Spec.Taints, m.Spec.Tolerations)
		if ok {
			toleratedNodes = append(toleratedNodes, node)
		}
	}

	selectedNodes := []corev1.Node{}

	// Filter the node list when a node selector is applied.
	if len(m.Spec.NodeSelectorTerms) > 0 {
		for _, node := range toleratedNodes {
			for _, term := range m.Spec.NodeSelectorTerms {
				for _, exp := range term.MatchExpressions {
					var ex selection.Operator

					// Convert the node selector operator into requirement
					// selection operator.
					switch exp.Operator {
					case corev1.NodeSelectorOpIn:
						ex = selection.Equals
					case corev1.NodeSelectorOpNotIn:
						ex = selection.NotEquals
					default:
						return "", fmt.Errorf("unsupported node selector term operator %q", exp.Operator)
					}

					// Create a new Requirement to perform label matching.
					req, err := labels.NewRequirement(exp.Key, ex, exp.Values)
					if err != nil {
						return "", fmt.Errorf("failed to create requirement: %v", err)
					}

					if req.Matches(labels.Set(node.GetLabels())) {
						selectedNodes = append(selectedNodes, node)
					}
				}
			}
		}
	} else {
		selectedNodes = toleratedNodes
	}

	// Log when node selector fails to select any node.
	if len(selectedNodes) == 0 {
		r.recorder.Event(m, corev1.EventTypeWarning, "FailedCreation", "no compatible nodes available for deployment, check node selector term and pod toleration options")
		log.WithValues("cluster", m.Name).Error(fmt.Errorf("no compatible nodes"), "No compatible nodes available for deployment of cluster")
	}

	nodeIPs := storageos.GetNodeIPs(selectedNodes)
	return strings.Join(nodeIPs, ","), nil
}

// Returns true and list of Tolerations matching all Taints if all are tolerated, or false otherwise.
// Taken from: https://github.com/kubernetes/kubernetes/blob/07a5488b2a8f67add543da72e8819407d8314204/pkg/apis/core/v1/helper/helpers.go#L426-L449
func getMatchingTolerations(taints []corev1.Taint, tolerations []corev1.Toleration) (bool, []corev1.Toleration) {
	if len(taints) == 0 {
		return true, []corev1.Toleration{}
	}
	if len(tolerations) == 0 && len(taints) > 0 {
		return false, []corev1.Toleration{}
	}
	result := []corev1.Toleration{}
	for i := range taints {
		tolerated := false
		for j := range tolerations {
			if tolerations[j].ToleratesTaint(&taints[i]) {
				result = append(result, tolerations[j])
				tolerated = true
				break
			}
		}
		if !tolerated {
			return false, []corev1.Toleration{}
		}
	}
	return true, result
}

// contains checks if an item exists in a given list.
func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// remove removes an item from a given list.
func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
