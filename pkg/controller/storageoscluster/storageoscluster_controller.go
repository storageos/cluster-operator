package storageoscluster

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	storageosv1alpha1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/pkg/util/k8sutil"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
)

// Add creates a new StorageOSCluster Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	// Get k8s version from client and set the version in ReconcileStorageOSCluster.
	clientset := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	version, err := k8sutil.GetK8SVersion(clientset)
	if err != nil {
		return err
	}
	log.Println("k8s version:", version)
	return add(mgr, newReconciler(mgr, strings.TrimLeft(version, "v")))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager, k8sVersion string) reconcile.Reconciler {
	return &ReconcileStorageOSCluster{
		client: mgr.GetClient(), scheme: mgr.GetScheme(), k8sVersion: k8sVersion, recorder: mgr.GetRecorder("storageoscluster-operator"),
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
	err = c.Watch(&source.Kind{Type: &storageosv1alpha1.StorageOSCluster{}}, &handler.EnqueueRequestForObject{})
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
	currentCluster *storageosv1alpha1.StorageOSCluster
	k8sVersion     string
	recorder       record.EventRecorder
}

// SetCurrentClusterIfNone checks if there's any existing current cluster and
// sets a new current cluster if it wasn't set before.
func (r *ReconcileStorageOSCluster) SetCurrentClusterIfNone(cluster *storageosv1alpha1.StorageOSCluster) {
	if r.currentCluster == nil {
		r.SetCurrentCluster(cluster)
	}
}

// SetCurrentCluster sets the currently active cluster in the controller.
func (r *ReconcileStorageOSCluster) SetCurrentCluster(cluster *storageosv1alpha1.StorageOSCluster) {
	r.currentCluster = cluster
}

// IsCurrentCluster compares a given cluster with the current cluster to check
// if they are the same.
func (r *ReconcileStorageOSCluster) IsCurrentCluster(cluster *storageosv1alpha1.StorageOSCluster) bool {
	if cluster == nil {
		return false
	}

	if (r.currentCluster.GetName() == cluster.GetName()) && (r.currentCluster.GetNamespace() == cluster.GetNamespace()) {
		return true
	}
	return false
}

// ResetCurrentCluster resets the current cluster of the controller.
func (r *ReconcileStorageOSCluster) ResetCurrentCluster() {
	// TODO: Remove cleanup at delete. This should never trigger automatically.
	// Users should trigger it explicitly via other means.
	// if r.currentCluster.Spec.CleanupAtDelete {
	// 	if err := cleanup(r.client, r.currentCluster); err != nil {
	// 		// This error is just logged and not returned. Failing to cleanup
	// 		// need not fail cluster reset.
	// 		log.Println(err)
	// 	}
	// }
	r.currentCluster = nil
}

// Reconcile reads that state of the cluster for a StorageOSCluster object and makes changes based on the state read
// and what is in the StorageOSCluster.Spec
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileStorageOSCluster) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// log.Printf("Reconciling StorageOSCluster %s/%s\n", request.Namespace, request.Name)

	reconcilePeriod := 15 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the StorageOSCluster instance
	instance := &storageosv1alpha1.StorageOSCluster{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Cluster instance not found. Reset the current cluster.
			r.ResetCurrentCluster()
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcileResult, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult, err
	}

	// Set as the current cluster if there's no current cluster.
	r.SetCurrentClusterIfNone(instance)

	// If the event doesn't belongs to the current cluster, do not reconcile.
	// There must be only a single instance of storageos in a cluster.
	if !r.IsCurrentCluster(instance) {
		err := fmt.Errorf("can't create more than one storageos cluster")
		r.recorder.Event(instance, corev1.EventTypeWarning, "FailedCreation", err.Error())
		return reconcileResult, err
	}

	if err := r.reconcile(instance); err != nil {
		return reconcileResult, err
	}

	return reconcileResult, nil
}

func (r *ReconcileStorageOSCluster) reconcile(m *storageosv1alpha1.StorageOSCluster) error {
	// Do not reconcile, the operator is paused for the cluster.
	if m.Spec.Pause {
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
	}

	// Update the spec values. This ensures that the default values are applied
	// when fields are not set in the spec.
	m.Spec.ResourceNS = m.Spec.GetResourceNS()
	m.Spec.Images.NodeContainer = m.Spec.GetNodeContainerImage()
	m.Spec.Images.InitContainer = m.Spec.GetInitContainerImage()
	m.Spec.Images.CleanupContainer = m.Spec.GetCleanupContainerImage()

	if m.Spec.CSI.Enable {
		m.Spec.Images.CSIDriverRegistrarContainer = m.Spec.GetCSIDriverRegistrarImage()
		m.Spec.Images.CSIExternalProvisionerContainer = m.Spec.GetCSIExternalProvisionerImage()
		m.Spec.Images.CSIExternalAttacherContainer = m.Spec.GetCSIExternalAttacherImage()
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
		stosDeployment := storageos.NewDeployment(r.client, m, r.recorder, r.scheme, r.k8sVersion)
		if err := stosDeployment.Deploy(); err != nil {
			// Ignore "Operation cannot be fulfilled" error. It happens when the
			// actual state of object is different from what is known to the operator.
			// Operator would resync and retry the failed operation on its own.
			if !strings.HasPrefix(err.Error(), "Operation cannot be fulfilled") {
				r.recorder.Event(m, corev1.EventTypeWarning, "FailedCreation", err.Error())
			}
			return err
		}
	} else {
		r.recorder.Event(m, corev1.EventTypeNormal, "Terminating", "StorageOS object deleted")
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
func (r *ReconcileStorageOSCluster) generateJoinToken(m *storageosv1alpha1.StorageOSCluster) (string, error) {
	// Get a new list of all the nodes.
	nodeList := storageos.NodeList()
	if err := r.client.List(context.Background(), &client.ListOptions{}, nodeList); err != nil {
		return "", fmt.Errorf("failed to list nodes: %v", err)
	}

	selectedNodes := []corev1.Node{}

	// Filter the node list when a node selector is applied.
	if len(m.Spec.NodeSelectorTerms) > 0 {
		for _, node := range nodeList.Items {
			// Skip a node with any taints. StorageOS pods don't support any
			// toleration.
			if len(node.Spec.Taints) > 0 {
				continue
			}
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
		selectedNodes = nodeList.Items
	}

	nodeIPs := storageos.GetNodeIPs(selectedNodes)
	return strings.Join(nodeIPs, ","), nil
}
