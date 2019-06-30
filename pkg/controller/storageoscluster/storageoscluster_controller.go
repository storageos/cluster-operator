package storageoscluster

import (
	"context"
	"fmt"
	"strings"
	"time"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	"github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/pkg/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("cluster")

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

	return add(mgr, newReconciler(mgr, strings.TrimLeft(version, "v")))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sVersion string) reconcile.Reconciler {
	return &ReconcileStorageOSCluster{
		client:     mgr.GetClient(),
		scheme:     mgr.GetScheme(),
		k8sVersion: k8sVersion,
		recorder:   mgr.GetRecorder("storageoscluster-operator"),
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
	client         client.Client
	scheme         *runtime.Scheme
	k8sVersion     string
	recorder       record.EventRecorder
	currentCluster *StorageOSCluster
}

// SetCurrentClusterIfNone checks if there's any existing current cluster and
// sets a new current cluster if it wasn't set before.
func (r *ReconcileStorageOSCluster) SetCurrentClusterIfNone(cluster *storageosv1.StorageOSCluster) {
	if r.currentCluster == nil {
		r.SetCurrentCluster(cluster)
	}
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

	reconcilePeriod := 15 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the StorageOSCluster instance
	instance := &storageosv1.StorageOSCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Cluster instance not found. Delete the resources and reset the
			// current cluster.
			if r.currentCluster != nil {
				if err := r.currentCluster.DeleteDeployment(); err != nil {
					// Error deleting - requeue the request.
					return reconcileResult, err
				}
			}
			r.ResetCurrentCluster()
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult, err
	}

	// Set as the current cluster if there's no current cluster.
	r.SetCurrentClusterIfNone(instance)

	// If the event doesn't belongs to the current cluster, do not reconcile.
	// There must be only a single instance of storageos in a cluster.
	if !r.currentCluster.IsCurrentCluster(instance) {
		err := fmt.Errorf("can't create more than one storageos cluster")
		r.recorder.Event(instance, corev1.EventTypeWarning, "FailedCreation", err.Error())
		return reconcileResult, err
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
		return reconcileResult, err
	}

	return reconcileResult, nil
}

func (r *ReconcileStorageOSCluster) reconcile(m *storageosv1.StorageOSCluster) error {
	if m.Spec.Pause {
		// Do not reconcile, the operator is paused for the cluster.
		return nil
	}

	join, err := r.generateJoinToken(m)
	if err != nil {
		return err
	}

	if m.Spec.Join != join {
		m.Spec.Join = join
		// Update Nodes as well, because updating StorageOS with null Nodes
		// results in invalid config.
		m.Status.Nodes = strings.Split(join, ",")
		if err := r.client.Update(context.Background(), m); err != nil {
			return err
		}
		// Update current cluster.
		r.SetCurrentCluster(m)
	}

	// Update the spec values. This ensures that the default values are applied
	// when fields are not set in the spec.
	m.Spec.ResourceNS = m.Spec.GetResourceNS()
	m.Spec.Images.NodeContainer = m.Spec.GetNodeContainerImage()
	m.Spec.Images.InitContainer = m.Spec.GetInitContainerImage()

	if m.Spec.CSI.Enable {
		m.Spec.Images.CSINodeDriverRegistrarContainer = m.Spec.GetCSINodeDriverRegistrarImage(storageos.CSIV1Supported(r.k8sVersion))
		if storageos.CSIV1Supported((r.k8sVersion)) {
			m.Spec.Images.CSIClusterDriverRegistrarContainer = m.Spec.GetCSIClusterDriverRegistrarImage()
			m.Spec.Images.CSILivenessProbeContainer = m.Spec.GetCSILivenessProbeImage()
		}
		m.Spec.Images.CSIExternalProvisionerContainer = m.Spec.GetCSIExternalProvisionerImage(storageos.CSIV1Supported(r.k8sVersion))
		m.Spec.Images.CSIExternalAttacherContainer = m.Spec.GetCSIExternalAttacherImage(storageos.CSIV1Supported(r.k8sVersion))
		m.Spec.CSI.DeploymentStrategy = m.Spec.GetCSIDeploymentStrategy()
	}

	if m.Spec.Ingress.Enable {
		m.Spec.Ingress.Hostname = m.Spec.GetIngressHostname()
	}

	m.Spec.Service.Name = m.Spec.GetServiceName()
	m.Spec.Service.Type = m.Spec.GetServiceType()
	m.Spec.Service.ExternalPort = m.Spec.GetServiceExternalPort()
	m.Spec.Service.InternalPort = m.Spec.GetServiceInternalPort()

	// Finalizers are set when an object should be deleted. Apply deploy only
	// when finalizers is empty.
	if len(m.GetFinalizers()) == 0 {
		// // Check if there's a new version of the cluster config and create a new
		// // deployment accordingly to update the resources that already exist.
		// // TODO: Add more granular checks. Resource version check is not enough
		// // to detect and apply changes. Maybe add an admission webhook to
		// // perform validation when the cluster config is updated and handle the
		// // resource update at an individual level. Updating all the resources
		// // is dangerous.
		// updateIfExists := false
		// if r.currentCluster.GetResourceVersion() != m.GetResourceVersion() {
		// 	log.Println("new cluster config detected")
		// 	updateIfExists = true
		// 	r.SetCurrentCluster(m)
		// }

		if err := r.currentCluster.Deploy(r); err != nil {
			// Ignore "Operation cannot be fulfilled" error. It happens when the
			// actual state of object is different from what is known to the operator.
			// Operator would resync and retry the failed operation on its own.
			if !strings.HasPrefix(err.Error(), "Operation cannot be fulfilled") {
				r.recorder.Event(m, corev1.EventTypeWarning, "FailedCreation", err.Error())
			}
			return err
		}
	} else {
		// Delete the deployment once the finalizers are set on the cluster
		// resource.
		r.recorder.Event(m, corev1.EventTypeNormal, "Terminating", "Deleting all the resources...")

		if err := r.currentCluster.DeleteDeployment(); err != nil {
			return err
		}

		r.ResetCurrentCluster()
		// Reset finalizers and let k8s delete the object.
		// When finalizers are set on an object, metadata.deletionTimestamp is
		// also set. deletionTimestamp helps the garbage collector identify
		// when to delete an object. k8s deletes the object only once the
		// list of finalizers is empty.
		m.SetFinalizers([]string{})
		return r.client.Update(context.Background(), m)
	}

	return nil
}

// generateJoinToken performs node selection based on NodeSelectorTerms if
// specified, and forms a join token by combining the node IPs.
func (r *ReconcileStorageOSCluster) generateJoinToken(m *storageosv1.StorageOSCluster) (string, error) {
	// Get a new list of all the nodes.
	nodeList := storageos.NodeList()
	if err := r.client.List(context.Background(), &client.ListOptions{}, nodeList); err != nil {
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
