package nfsserver

import (
	"context"
	goerrors "errors"
	"strings"
	"time"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	stosClientset "github.com/storageos/cluster-operator/pkg/client/clientset/versioned"
	"github.com/storageos/cluster-operator/pkg/nfs"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// ErrNoCluster is the error when there's no associated running StorageOS
// cluster found for NFS server.
var ErrNoCluster = goerrors.New("no storageos cluster found")

var log = logf.Log.WithName("controller_nfsserver")

const finalizer = "finalizer.nfsserver.storageos.com"

// Add creates a new NFSServer Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	storageos := stosClientset.NewForConfigOrDie(mgr.GetConfig())
	return &ReconcileNFSServer{
		client:        mgr.GetClient(),
		scheme:        mgr.GetScheme(),
		recorder:      mgr.GetRecorder("storageos-nfsserver"),
		stosClientset: storageos,
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("nfsserver-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NFSServer
	err = c.Watch(&source.Kind{Type: &storageosv1.NFSServer{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource NFSServer.
	err = c.Watch(&source.Kind{Type: &storageosv1.NFSServer{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource StatefulSet and requeue the owner
	// NFSServer.
	err = c.Watch(&source.Kind{Type: &appsv1.StatefulSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &storageosv1.NFSServer{},
	})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource Service and requeue the owner
	// NFSServer.
	//
	// This is used to update the NFSServer Status with the connection endpoint
	// once it comes online.
	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &storageosv1.NFSServer{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileNFSServer implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileNFSServer{}

// ReconcileNFSServer reconciles a NFSServer object
type ReconcileNFSServer struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client        client.Client
	stosClientset stosClientset.Interface
	scheme        *runtime.Scheme
	recorder      record.EventRecorder
}

// Reconcile reads that state of the cluster for a NFSServer object and makes changes based on the state read
// and what is in the NFSServer.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileNFSServer) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	// reqLogger.Info("Reconciling NFSServer")

	reconcilePeriod := 15 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the NFSServer instance
	instance := &storageosv1.NFSServer{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult, err
	}

	if err := r.reconcile(instance); err != nil {
		reqLogger.V(4).Info("Reconcile failed", "error", err)
		return reconcileResult, err
	}

	return reconcileResult, nil
}

func (r *ReconcileNFSServer) reconcile(instance *storageosv1.NFSServer) error {
	// Add our finalizer immediately so we can cleanup a partial deployment.  If
	// this is not set, the CR can simply be deleted.
	if len(instance.GetFinalizers()) == 0 {

		// Add our finalizer so that we control deletion.
		if err := r.addFinalizer(instance); err != nil {
			return err
		}

		// Return here, as the update to add the finalizer will trigger another
		// reconcile.
		return nil
	}

	// Get a StorageOS cluster to associate the NFS server with.
	stosCluster, err := r.getCurrentStorageOSCluster()
	if err != nil {
		return err
	}

	// Update NFS spec with values inferred from the StorageOS cluster.
	instance.Spec.StorageClassName = instance.Spec.GetStorageClassName(stosCluster.Spec.GetStorageClassName())

	// Prepare for NFS deployment.

	// Labels to be applied on all the k8s resources that are created for NFS
	// server. Inherit the labels from the CR.
	labels := instance.Labels
	if labels == nil {
		labels = map[string]string{}
	}
	// Add default labels.
	labels["app"] = "storageos"

	d := nfs.NewDeployment(r.client, stosCluster, instance, labels, r.recorder, r.scheme)

	// If the CR has not been marked for deletion, ensure it is deployed.
	if instance.GetDeletionTimestamp() == nil {
		if err := d.Deploy(); err != nil {
			// Ignore "Operation cannot be fulfilled" error. It happens when the
			// actual state of object is different from what is known to the operator.
			// Operator would resync and retry the failed operation on its own.
			if !strings.HasPrefix(err.Error(), "Operation cannot be fulfilled") {
				r.recorder.Event(instance, corev1.EventTypeWarning, "FailedCreation", err.Error())
			}
			return err
		}
	} else {
		// Delete the deployment once the finalizers are set on the cluster
		// resource.
		r.recorder.Event(instance, corev1.EventTypeNormal, "Terminating", "Deleting the NFS server.")

		if err := d.Delete(); err != nil {
			return err
		}

		// Reset finalizers and let k8s delete the object.
		// When finalizers are set on an object, metadata.deletionTimestamp is
		// also set. deletionTimestamp helps the garbage collector identify
		// when to delete an object. k8s deletes the object only once the
		// list of finalizers is empty.
		instance.SetFinalizers([]string{})
		return r.client.Update(context.Background(), instance)
	}

	return nil
}

func (r *ReconcileNFSServer) addFinalizer(instance *storageosv1.NFSServer) error {

	instance.SetFinalizers(append(instance.GetFinalizers(), finalizer))

	// Update CR
	err := r.client.Update(context.TODO(), instance)
	if err != nil {
		return err
	}
	return nil
}

// getCurrentStorageOSCluster returns a running StorageOS cluster.
func (r *ReconcileNFSServer) getCurrentStorageOSCluster() (*storageosv1.StorageOSCluster, error) {
	var currentCluster *storageosv1.StorageOSCluster

	// Get a list of all the StorageOS clusters.
	stosClusters, err := r.stosClientset.StorageosV1().StorageOSClusters("").List(metav1.ListOptions{})
	if err != nil {
		return currentCluster, err
	}

	for _, cluster := range stosClusters.Items {
		// Only one cluster can be in running phase at a time.
		if cluster.Status.Phase == storageosv1.ClusterPhaseRunning {
			currentCluster = &cluster
			break
		}
	}

	// If no current cluster found, fail.
	if currentCluster != nil {
		return currentCluster, nil
	}

	return currentCluster, ErrNoCluster
}

func contains(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

func remove(list []string, s string) []string {
	for i, v := range list {
		if v == s {
			list = append(list[:i], list[i+1:]...)
		}
	}
	return list
}
