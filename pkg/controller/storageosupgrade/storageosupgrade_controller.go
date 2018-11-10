package storageosupgrade

import (
	"context"
	"fmt"
	"time"

	storageosv1alpha1 "github.com/storageos/cluster-operator/pkg/apis/storageos/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var (
	// operatorImage is the image name of controller-operator. This is needed
	// because the upgrader binary is baked into the same cluster-operator image.
	// This is set at build time using linker flags to be the same as build
	// container image name.
	operatorImage string
)

// Add creates a new StorageOSUpgrade Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileStorageOSUpgrade{client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetRecorder("storageos-upgrader")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("storageosupgrade-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource StorageOSUpgrade
	err = c.Watch(&source.Kind{Type: &storageosv1alpha1.StorageOSUpgrade{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileStorageOSUpgrade{}

// ReconcileStorageOSUpgrade reconciles a StorageOSUpgrade object
type ReconcileStorageOSUpgrade struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client         client.Client
	scheme         *runtime.Scheme
	currentUpgrade *storageosv1alpha1.StorageOSUpgrade
	recorder       record.EventRecorder
}

// SetCurrentUpgradeIfNone checks if there's any existing current upgrade and
// sets a new current upgrade if it wasn't set before.
func (r *ReconcileStorageOSUpgrade) SetCurrentUpgradeIfNone(upgrade *storageosv1alpha1.StorageOSUpgrade) {
	if r.currentUpgrade == nil {
		r.SetCurrentUpgrade(upgrade)
	}
}

// SetCurrentUpgrade sets the currently active upgrade in the controller.
func (r *ReconcileStorageOSUpgrade) SetCurrentUpgrade(upgrade *storageosv1alpha1.StorageOSUpgrade) {
	r.currentUpgrade = upgrade
}

// IsCurrentUpgrade compares a given upgrade with the current cluster to check
// if they are the same.
func (r *ReconcileStorageOSUpgrade) IsCurrentUpgrade(upgrade *storageosv1alpha1.StorageOSUpgrade) bool {
	if upgrade == nil {
		return false
	}

	if (r.currentUpgrade.GetName() == upgrade.GetName()) && (r.currentUpgrade.GetNamespace() == upgrade.GetNamespace()) {
		return true
	}
	return false
}

// ResetCurrentUpgrade resets the current upgrade of the controller.
func (r *ReconcileStorageOSUpgrade) ResetCurrentUpgrade() {
	r.currentUpgrade = nil
}

// Reconcile reads that state of the cluster for a StorageOSUpgrade object and makes changes based on the state read
// and what is in the StorageOSUpgrade.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileStorageOSUpgrade) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// log.Printf("Reconciling StorageOSUpgrade %s/%s\n", request.Namespace, request.Name)

	reconcilePeriod := 10 * time.Second
	reconcileResult := reconcile.Result{RequeueAfter: reconcilePeriod}

	// Fetch the StorageOSUpgrade instance
	instance := &storageosv1alpha1.StorageOSUpgrade{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Upgrade instance not found. Reset the current cluster.
			r.ResetCurrentUpgrade()
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcileResult, err
	}

	r.SetCurrentUpgradeIfNone(instance)

	if !r.IsCurrentUpgrade(instance) {
		err := fmt.Errorf("can't create more than one storageos upgrade")
		r.recorder.Event(instance, corev1.EventTypeWarning, "FailedCreation", err.Error())
		return reconcileResult, err
	}

	// Create a ServiceAccount for the upgrader.
	sa := newServiceAccountForCR("storageos-upgrader-sa", instance)
	if err := controllerutil.SetControllerReference(instance, sa, r.scheme); err != nil {
		return reconcileResult, err
	}
	if err := r.client.Create(context.Background(), sa); err != nil && !errors.IsAlreadyExists(err) {
		return reconcileResult, fmt.Errorf("failed to create service account: %v", err)
	}

	// Create a ClusterRole for the upgrader.
	// This must cluster scoped because the applications can be in any namespace.
	rules := []rbacv1.PolicyRule{
		{
			APIGroups: []string{"apps"},
			Resources: []string{"daemonsets", "deployments", "statefulsets"},
			Verbs:     []string{"get", "list", "update", "patch"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"get", "list"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"persistentvolumeclaims"},
			Verbs:     []string{"get", "list"},
		},
		{
			APIGroups: []string{"storage.k8s.io"},
			Resources: []string{"storageclasses"},
			Verbs:     []string{"get", "list"},
		},
	}
	cr := newClusterRole("storageos-upgrader-role", rules)
	if err := controllerutil.SetControllerReference(instance, cr, r.scheme); err != nil {
		return reconcileResult, err
	}
	if err := r.client.Create(context.Background(), cr); err != nil && !errors.IsAlreadyExists(err) {
		return reconcileResult, fmt.Errorf("failed to create cluster role: %v", err)
	}

	// Create ClusterRoleBinding for the ServiceAccount.
	subjects := []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      "storageos-upgrader-sa",
			Namespace: instance.GetNamespace(),
		},
	}
	roleRef := rbacv1.RoleRef{
		Kind:     "ClusterRole",
		Name:     cr.GetName(),
		APIGroup: rbacv1.GroupName,
	}
	crb := newClusterRoleBinding("storageos-upgrader-clusterrolebinding", subjects, roleRef)
	if err := controllerutil.SetControllerReference(instance, crb, r.scheme); err != nil {
		return reconcileResult, err
	}
	if err := r.client.Create(context.Background(), crb); err != nil && !errors.IsAlreadyExists(err) {
		return reconcileResult, fmt.Errorf("failed to create cluster role binding: %v", err)
	}

	// Define a new Job object.
	job := newJobForCR(instance)

	// Set StorageOSUpgrade instance as the owner and controller.
	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return reconcileResult, err
	}

	// Check if this Job already exists.
	found := &batchv1.Job{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		err = r.client.Create(context.TODO(), job)
		if err != nil {
			return reconcileResult, err
		}

		// Job created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcileResult, err
	}

	if r.currentUpgrade.Status.Completed {
		// Upgrade completed. Do nothing.
		return reconcileResult, nil
	}

	if found.Status.Succeeded == 1 {
		r.currentUpgrade.Status.Completed = true
		successMessage := fmt.Sprintf("StorageOS upgraded to %s", instance.Spec.NewImage)
		r.recorder.Event(instance, corev1.EventTypeNormal, "UpgradeComplete", successMessage)
	}

	return reconcileResult, nil
}

// newJobForCR returns a job with the same name/namespace as the cr.
func newJobForCR(cr *storageosv1alpha1.StorageOSUpgrade) *batchv1.Job {
	labels := map[string]string{
		"app": cr.Name,
	}

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-job",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					ServiceAccountName: "storageos-upgrader-sa",
					Containers: []corev1.Container{
						{
							Image:   operatorImage,
							Name:    "storageos-upgrader",
							Command: []string{"upgrader"},
							Env: []corev1.EnvVar{
								{
									Name:  "NEW_IMAGE",
									Value: cr.Spec.NewImage,
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
		},
	}
}

func newServiceAccountForCR(name string, cr *storageosv1alpha1.StorageOSUpgrade) *corev1.ServiceAccount {
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: cr.Namespace,
			Labels: map[string]string{
				"app": "storageos-upgrader",
			},
		},
	}
}

func newClusterRoleBinding(name string, subjects []rbacv1.Subject, roleRef rbacv1.RoleRef) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "storageos-upgrader",
			},
		},
		Subjects: subjects,
		RoleRef:  roleRef,
	}
}

func newClusterRole(name string, rules []rbacv1.PolicyRule) *rbacv1.ClusterRole {
	return &rbacv1.ClusterRole{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "rbac.authorization.k8s.io/v1",
			Kind:       "ClusterRole",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "storageos-upgrader",
			},
		},
		Rules: rules,
	}
}
