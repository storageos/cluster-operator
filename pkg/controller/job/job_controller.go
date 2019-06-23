package job

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	storageosv1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = ctrl.Log.WithName("job")

// Add creates a new Job Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	clientset := kubernetes.NewForConfigOrDie(mgr.GetConfig())
	return &ReconcileJob{client: mgr.GetClient(), scheme: mgr.GetScheme(), clientset: clientset, recorder: mgr.GetRecorder("storageoscluster-operator")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("job-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Job
	err = c.Watch(&source.Kind{Type: &storageosv1.Job{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to secondary resource DaemonSet and requeue the owner Job
	err = c.Watch(&source.Kind{Type: &appsv1.DaemonSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &storageosv1.Job{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileJob{}

// ReconcileJob reconciles a Job object
type ReconcileJob struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client    client.Client
	clientset kubernetes.Interface
	scheme    *runtime.Scheme
	recorder  record.EventRecorder
}

// Reconcile reads that state of the cluster for a Job object and makes changes based on the state read
// and what is in the Job.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileJob) Reconcile(request reconcile.Request) (reconcile.Result, error) {

	log := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	// log.Info("Reconciling Job")

	reconcilePeriod := 10 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the Job instance
	instance := &storageosv1.Job{}
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

	// Set Spec attribute values.
	instance.Spec.LabelSelector = instance.Spec.GetLabelSelector()

	// Update the object.
	if err := r.client.Update(context.Background(), instance); err != nil {
		return reconcileResult, err
	}

	// Define a new DaemonSet object
	daemonset, err := newDaemonSetForCR(instance)
	if err != nil {
		return reconcileResult, err
	}

	// Set Job instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, daemonset, r.scheme); err != nil {
		return reconcileResult, err
	}

	// Check if this DaemonSet already exists
	found := &appsv1.DaemonSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: daemonset.Name, Namespace: daemonset.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		log.Info("creating a new DaemonSet")
		err = r.client.Create(context.TODO(), daemonset)
		if err != nil {
			return reconcileResult, err
		}

		// DaemonSet created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcileResult, err
	}

	if instance.Status.Completed {
		// Job completed. Do nothing - don't requeue
		return reconcile.Result{}, nil
	}

	// Check DaemonSet Pod status.
	completed, err := checkPods(r.clientset, instance, r.recorder)
	if err != nil {
		return reconcileResult, err
	}

	// Update the Completed status of the Job.
	instance.Status.Completed = completed
	if err := r.client.Update(context.Background(), instance); err != nil {
		return reconcileResult, err
	}

	return reconcileResult, nil
}

// checkPods checks the logs of all pods with the given label selector for the
// completionWord and published a "JobCompleted" event when all the pods have
// completed their task.
func checkPods(client kubernetes.Interface, cr *storageosv1.Job, recorder record.EventRecorder) (bool, error) {
	podListOpts := metav1.ListOptions{
		LabelSelector: cr.Spec.GetLabelSelector(),
	}

	pods, err := client.CoreV1().Pods(cr.GetNamespace()).List(podListOpts)
	if err != nil {
		log.Error(err, "failed to get podList")
		return false, err
	}

	totalPods := len(pods.Items)
	completedPods := 0

	// Skip if there are no daemonset-job pods.
	if totalPods == 0 {
		log.Info("no DaemonSets found")
		return false, nil
	}

	opts := &corev1.PodLogOptions{}

	for _, p := range pods.Items {
		req := client.CoreV1().Pods(p.GetNamespace()).GetLogs(p.GetName(), opts)
		logText, err := getPlainLogs(req)
		if err != nil {
			log.Error(err, "failed to get logs from pod", "pod", p.GetName())
			// Continue checking other pods.
			continue
		}

		if strings.Contains(logText, cr.Spec.CompletionWord) {
			completedPods++
		}
	}

	if totalPods == completedPods {
		recorder.Event(cr, corev1.EventTypeNormal, "JobCompleted", "Job Completed. Safe to delete.")
		return true, nil
	}

	return false, nil
}

// getPlainLogs reads the logs from a request and returns the log text as string.
func getPlainLogs(req *restclient.Request) (string, error) {
	var buf bytes.Buffer
	readCloser, err := req.Stream()
	if err != nil {
		return "", err
	}

	defer readCloser.Close()
	_, err = io.Copy(&buf, readCloser)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newDaemonSetForCR(cr *storageosv1.Job) (*appsv1.DaemonSet, error) {
	defaultLabels := map[string]string{
		"daemonset": cr.Name + "-daemonset-job",
		"job":       cr.Name,
	}

	selectorMap, err := labels.ConvertSelectorToLabelsMap(cr.Spec.GetLabelSelector())
	if err != nil {
		return nil, err
	}
	// Merge the default labels with the job label selector.
	// The label selector labels must be present in all the DaemonSet Pods.
	mergedLabels := labels.Merge(defaultLabels, selectorMap)

	dset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-daemonset-job",
			Namespace: cr.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: mergedLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mergedLabels,
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "job-container",
							Image: cr.Spec.Image,
							Args:  cr.Spec.Args,
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "target",
									MountPath: cr.Spec.MountPath,
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "target",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: cr.Spec.HostPath,
								},
							},
						},
					},
				},
			},
		},
	}

	// Add pod tolerations if defined.
	tolerations := cr.Spec.Tolerations
	for i := range tolerations {
		if tolerations[i].Operator == corev1.TolerationOpExists && tolerations[i].Value != "" {
			return nil, fmt.Errorf("key(%s): toleration value must be empty when `operator` is 'Exists'", tolerations[i].Key)
		}
	}
	if len(tolerations) > 0 {
		dset.Spec.Template.Spec.Tolerations = cr.Spec.Tolerations
	}

	// Add node affinity if defined.
	if len(cr.Spec.NodeSelectorTerms) > 0 {
		dset.Spec.Template.Spec.Affinity = &corev1.Affinity{NodeAffinity: &corev1.NodeAffinity{
			RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
				NodeSelectorTerms: cr.Spec.NodeSelectorTerms,
			},
		}}
	}

	return dset, nil
}
