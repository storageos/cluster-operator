package k8sutil

import (
	"fmt"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"github.com/storageos/cluster-operator/pkg/util/task"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

const (
	pvcStorageClassKey            = "volume.beta.kubernetes.io/storage-class"
	storageosProvisioner          = "storageos"
	replicaKey                    = "stos/replicas-before-scale-down"
	deploymentUpdateTimeout       = 5 * time.Minute
	statefulSetUpdateTimeout      = 10 * time.Minute
	defaultRetryInterval          = 10 * time.Second
	daemonsetUpdateTriggerTimeout = 5 * time.Minute
)

// K8SOps is a kubernetes operations type which can be used to query and perform
// actions on a kubernetes cluster.
type K8SOps struct {
	client kubernetes.Interface
	logger logr.Logger
}

// NewK8SOps creates and returns a new k8sOps object.
func NewK8SOps(client kubernetes.Interface, logger logr.Logger) *K8SOps {
	return &K8SOps{
		client: client,
		logger: logger,
	}
}

// GetK8SVersion queries and returns kubernetes server version.
func (k K8SOps) GetK8SVersion() (string, error) {
	info, err := k.client.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return info.String(), nil
}

// EventRecorder creates and returns an EventRecorder which could be used to
// broadcast events for k8s objects.
func (k K8SOps) EventRecorder() record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: k.client.CoreV1().Events(""),
		},
	)
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: "storageoscluster-operator"},
	)
	return recorder
}

// GetDeploymentsUsingStorageClassProvisioner returns a DeploymentList that use a given
// StorageClass name in the PVC.
func (k K8SOps) GetDeploymentsUsingStorageClassProvisioner(provisionerName string) (*appsv1.DeploymentList, error) {
	deployments, err := k.client.AppsV1().Deployments("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	scDeployments := &appsv1.DeploymentList{}
	for _, dep := range deployments.Items {
		for _, v := range dep.Spec.Template.Spec.Volumes {
			if v.PersistentVolumeClaim == nil {
				continue
			}

			pvc, err := k.client.CoreV1().PersistentVolumeClaims(dep.GetNamespace()).Get(v.PersistentVolumeClaim.ClaimName, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					continue
				}
				return nil, err
			}

			sc, err := k.GetStorageClassForPVC(pvc)
			if err == nil && sc.Provisioner == provisionerName {
				scDeployments.Items = append(scDeployments.Items, dep)
				break
			}
		}
	}

	return scDeployments, nil
}

// GetStatefulSetsUsingStorageClassProvisioner returns StatefulSets using PVC
// with a given provisioner.
func (k K8SOps) GetStatefulSetsUsingStorageClassProvisioner(provisionerName string) (*appsv1.StatefulSetList, error) {
	ss, err := k.client.AppsV1().StatefulSets("").List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	scStatefulsets := &appsv1.StatefulSetList{}
	for _, s := range ss.Items {
		if s.Spec.VolumeClaimTemplates == nil {
			continue
		}

		for _, template := range s.Spec.VolumeClaimTemplates {
			sc, err := k.GetStorageClassForPVC(&template)
			if err == nil && sc.Provisioner == provisionerName {
				scStatefulsets.Items = append(scStatefulsets.Items, s)
				break
			}
			return nil, err
		}
	}

	return scStatefulsets, nil
}

// GetStorageClassForPVC returns storage class for a given PVC.
func (k K8SOps) GetStorageClassForPVC(pvc *corev1.PersistentVolumeClaim) (*storagev1.StorageClass, error) {
	var scName string
	if pvc.Spec.StorageClassName != nil && len(*pvc.Spec.StorageClassName) > 0 {
		scName = *pvc.Spec.StorageClassName
	} else {
		scName = pvc.Annotations[pvcStorageClassKey]
	}

	if len(scName) == 0 {
		return nil, fmt.Errorf("PVC: %s does not have a storage class", pvc.Name)
	}

	return k.client.StorageV1().StorageClasses().Get(scName, metav1.GetOptions{})
}

// GetStorageOSApps returns a DeploymentList and a StatefulSetList of the apps
// using StorageOS PVC.
func (k K8SOps) GetStorageOSApps() (*appsv1.DeploymentList, *appsv1.StatefulSetList, error) {
	stosDeployments, err := k.GetDeploymentsUsingStorageClassProvisioner(storageosProvisioner)
	if err != nil {
		return nil, nil, err
	}

	stosStatefulSets, err := k.GetStatefulSetsUsingStorageClassProvisioner(storageosProvisioner)
	if err != nil {
		return nil, nil, err
	}

	return stosDeployments, stosStatefulSets, nil
}

// ScaleDownApps fetches all the applications running storageos and scales down
// their replica count to zero.
func (k K8SOps) ScaleDownApps() error {
	deps, ss, err := k.GetStorageOSApps()
	if err != nil {
		return err
	}

	var valZero int32
	for _, d := range deps.Items {
		k.logger.WithValues("Namespace", d.GetNamespace(), "Name", d.GetName()).Info("scaling down deployment")

		t := func() (interface{}, bool, error) {
			dCopy, err := k.client.AppsV1().Deployments(d.GetNamespace()).Get(d.GetName(), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil, false, nil
				}

				return nil, true, err
			}

			if *dCopy.Spec.Replicas == 0 {
				k.logger.WithValues("Namespace", dCopy.Namespace, "Name", dCopy.Name).Info("app already scaled down to 0")
				return nil, false, nil
			}

			dCopy.Annotations[replicaKey] = fmt.Sprintf("%d", *dCopy.Spec.Replicas)
			dCopy.Spec.Replicas = &valZero
			_, updateErr := k.client.AppsV1().Deployments(d.GetNamespace()).Update(dCopy)
			if updateErr != nil {
				k.logger.Error(updateErr, "failed to update Deployment", "Name", dCopy.GetName())
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, deploymentUpdateTimeout, defaultRetryInterval, k.logger); err != nil {
			return err
		}
	}

	for _, s := range ss.Items {
		k.logger.WithValues("Namespace", s.GetNamespace(), "Name", s.GetName()).Info("scaling down statefulset")

		t := func() (interface{}, bool, error) {
			sCopy, err := k.client.AppsV1().StatefulSets(s.GetNamespace()).Get(s.GetName(), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil, false, nil
				}

				return nil, true, err
			}

			if *sCopy.Spec.Replicas == 0 {
				k.logger.WithValues("Namespace", sCopy.Namespace, "Name", sCopy.Name).Info("app already scaled down to 0")
				return nil, false, nil
			}

			sCopy.Annotations[replicaKey] = fmt.Sprintf("%d", *sCopy.Spec.Replicas)
			sCopy.Spec.Replicas = &valZero
			_, updateErr := k.client.AppsV1().StatefulSets(s.GetNamespace()).Get(s.GetName(), metav1.GetOptions{})
			if updateErr != nil {
				k.logger.Error(updateErr, "failed to update StatefulSet", "Name", sCopy.GetName())
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, statefulSetUpdateTimeout, defaultRetryInterval, k.logger); err != nil {
			return err
		}
	}

	return nil
}

// ScaleUpApps fetches all the applications running storageos and scales up
// their replica count to their previous values.
func (k K8SOps) ScaleUpApps() error {
	deps, ss, err := k.GetStorageOSApps()
	if err != nil {
		return err
	}

	for _, d := range deps.Items {
		k.logger.WithValues("Namespace", d.Namespace, "Name", d.Name).Info("restoring app")

		t := func() (interface{}, bool, error) {
			dCopy, err := k.client.AppsV1().Deployments(d.GetNamespace()).Get(d.GetName(), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil, false, nil
				}

				return nil, true, err
			}

			if dCopy.Annotations == nil {
				// This app wasn't scaled down by the updater.
				return nil, false, nil
			}

			val, present := dCopy.Annotations[replicaKey]
			if !present || len(val) == 0 {
				k.logger.WithValues("Namespace", dCopy.GetNamespace(), "Name", dCopy.GetName()).Info("not restoring app: no annotation found to track replica count")
				return nil, false, nil
			}

			i, err := strconv.Atoi(val)
			if err != nil {
				k.logger.Error(err, "failed to read replica", "Name", dCopy.GetName())
				return nil, false, nil
			}

			delete(dCopy.Annotations, replicaKey)
			dCopy.Spec.Replicas = int32Ptr(int32(i))
			_, updateErr := k.client.AppsV1().Deployments(dCopy.GetNamespace()).Update(dCopy)
			if updateErr != nil {
				k.logger.Error(updateErr, "failed to update DaemonSet", "Name", dCopy.GetName())
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, deploymentUpdateTimeout, defaultRetryInterval, k.logger); err != nil {
			return err
		}
	}

	for _, s := range ss.Items {
		k.logger.WithValues("Namespace", s.Namespace, "Name", s.Name).Info("restoring app")

		t := func() (interface{}, bool, error) {
			sCopy, err := k.client.AppsV1().StatefulSets(s.GetNamespace()).Get(s.GetName(), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil, false, nil
				}

				return nil, true, err
			}

			if sCopy.Annotations == nil {
				// This app wasn't scaled down by the upgrader.
				return nil, false, nil
			}

			val, present := sCopy.Annotations[replicaKey]
			if !present || len(val) == 0 {
				k.logger.WithValues("Namespace", sCopy.GetNamespace(), "Name", sCopy.GetName()).Info("not restoring app: no annotation found to track replica count")
				return nil, false, nil
			}

			i, err := strconv.Atoi(val)
			if err != nil {
				k.logger.Error(err, "failed to read replica", "Name", sCopy.GetName())
				return nil, false, nil
			}

			delete(sCopy.Annotations, replicaKey)
			sCopy.Spec.Replicas = int32Ptr(int32(i))
			_, updateErr := k.client.AppsV1().StatefulSets(sCopy.GetNamespace()).Update(sCopy)
			if updateErr != nil {
				k.logger.Error(updateErr, "failed to update StatefulSet", "Name", sCopy.GetName())
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, statefulSetUpdateTimeout, defaultRetryInterval, k.logger); err != nil {
			return err
		}

	}

	return nil
}

// UpgradeDaemonSet upgrades the storageos daemonsets with a new container
// image. It blocks and waits checking the status of pods before exiting. Once
// all the pods are ready, it exits.
func (k K8SOps) UpgradeDaemonSet(newImage string) error {
	ds, err := k.GetStorageOSDaemonSet()
	if err != nil {
		return err
	}

	k.logger.WithValues("Namespace", ds.GetNamespace(), "Name", ds.GetName()).Info("updating storageos daemonset")

	expectedGenerations := make(map[types.UID]int64)

	t := func() (interface{}, bool, error) {
		dCopy, err := k.client.AppsV1().DaemonSets(ds.GetNamespace()).Get(ds.GetName(), metav1.GetOptions{})
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, false, nil
			}

			return nil, true, err
		}

		podList, err := k.GetDaemonSetPods(dCopy)
		if err != nil {
			return nil, true, err
		}

		// totalDaemonSetPods should be set as the MaxUnavailable in
		// RollingUpdate so that all the current generation pods are terminated
		// together and the new generation pods are created together.
		totalDaemonSetPods := len(podList.Items)

		// Save and use ObservedGeneration + 1 to figure out the currently
		// applied config of a daemonset.
		expectedGenerations[dCopy.GetUID()] = dCopy.Status.ObservedGeneration + 1

		// Set the DaemonSet update strategy.
		dCopy.Spec.UpdateStrategy = appsv1.DaemonSetUpdateStrategy{
			Type: appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &appsv1.RollingUpdateDaemonSet{
				MaxUnavailable: &intstr.IntOrString{Type: intstr.Int, IntVal: int32(totalDaemonSetPods)},
			},
		}
		// Set the new container image.
		dCopy.Spec.Template.Spec.Containers[0].Image = newImage
		_, updateErr := k.client.AppsV1().DaemonSets(dCopy.GetNamespace()).Update(dCopy)
		if updateErr != nil {
			k.logger.Error(updateErr, "failed to update DaemonSet", "Name", dCopy.GetName())
			return nil, true, updateErr
		}

		return nil, false, nil
	}

	if _, err := task.DoRetryWithTimeout(t, deploymentUpdateTimeout, defaultRetryInterval, k.logger); err != nil {
		return err
	}

	// Wait for the new daemonset to be ready
	k.logger.WithValues("Namespace", ds.GetNamespace(), "Name", ds.GetName(), "image", newImage).Info("checking upgrade status of DaemonSet")

	// Check the DaemonSet generation and block until the latest expected
	// generation is available.
	t = func() (interface{}, bool, error) {
		updatedDS, err := k.client.AppsV1().DaemonSets(ds.GetNamespace()).Get(ds.GetName(), metav1.GetOptions{})
		if err != nil {
			return nil, true, err
		}

		expectedGeneration, _ := expectedGenerations[ds.UID]
		if updatedDS.Status.ObservedGeneration != expectedGeneration {
			return nil, true, fmt.Errorf("daemonset: [%s] %s still running previous generation: %d. Expected generation %d", ds.GetNamespace(), ds.GetName(), updatedDS.Status.ObservedGeneration, expectedGeneration)
		}

		return nil, false, nil
	}

	if _, err := task.DoRetryWithTimeout(t, daemonsetUpdateTriggerTimeout, defaultRetryInterval, k.logger); err != nil {
		return err
	}

	k.logger.Info("Waiting for the daemonset to be ready")

	if err = k.WaitForDaemonSetToBeReady(ds.GetName(), ds.GetNamespace()); err != nil {
		return err
	}

	return nil
}

// WaitForDaemonSetToBeReady checks a given DaemonSet to be available and ready.
func (k K8SOps) WaitForDaemonSetToBeReady(name, namespace string) error {
	t := func() (interface{}, bool, error) {
		ds, err := k.client.AppsV1().DaemonSets(namespace).Get(name, metav1.GetOptions{})
		if err != nil {
			return nil, true, err
		}

		if ds.Status.ObservedGeneration == 0 {
			return nil, true, fmt.Errorf("Observed generation is still 0. Check after some time")
		}

		pods, err := k.GetDaemonSetPods(ds)
		if err != nil {
			return nil, true, fmt.Errorf("Failed to get daemonset pods")
		}

		if len(pods.Items) == 0 {
			return nil, true, fmt.Errorf("Daemonset has 0 pods")
		}

		if ds.Status.DesiredNumberScheduled != ds.Status.UpdatedNumberScheduled {
			return nil, true, fmt.Errorf("Not all pods are updated")
		}

		if ds.Status.NumberUnavailable > 0 {
			return nil, true, fmt.Errorf("%d pods are unavailable", ds.Status.NumberUnavailable)
		}

		return nil, false, nil
	}

	if _, err := task.DoRetryWithTimeout(t, daemonsetUpdateTriggerTimeout, defaultRetryInterval, k.logger); err != nil {
		return err
	}

	return nil
}

// GetStorageOSDaemonSet returns a DaemonSet that runs storageos.
func (k K8SOps) GetStorageOSDaemonSet() (*appsv1.DaemonSet, error) {
	dss, err := k.GetDaemonSetsByLabel("app=storageos")
	if err != nil {
		return nil, err
	}

	if len(dss.Items) == 0 {
		return nil, fmt.Errorf("could not find any storageos daemonset")
	}

	if len(dss.Items) > 1 {
		return nil, fmt.Errorf("can't upgrade, found more than one storageos daemonset")
	}

	return &dss.Items[0], nil
}

// GetDaemonSetsByLabel returns DaemonSets selected by the given label.
func (k K8SOps) GetDaemonSetsByLabel(label string) (*appsv1.DaemonSetList, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: label,
	}

	dss, err := k.client.AppsV1().DaemonSets("").List(listOpts)
	if err != nil {
		return nil, err
	}

	return dss, nil
}

// GetDaemonSetPods return PodList of all the pods that belong to a given
// DaemonSet.
func (k K8SOps) GetDaemonSetPods(ds *appsv1.DaemonSet) (*corev1.PodList, error) {
	return k.GetPodsByOwner(ds.GetUID(), ds.GetNamespace())
}

// GetPodsByOwner returns PodList of all the pods that are owned by the given
// ownerUID in the given namespace.
func (k K8SOps) GetPodsByOwner(ownerUID types.UID, namespace string) (*corev1.PodList, error) {
	pods, err := k.client.CoreV1().Pods(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := &corev1.PodList{}
	for _, pod := range pods.Items {
		for _, owner := range pod.OwnerReferences {
			if owner.UID == ownerUID {
				result.Items = append(result.Items, pod)
			}
		}
	}

	if len(result.Items) == 0 {
		return nil, fmt.Errorf("pods with not found")
	}

	return result, nil
}

func int32Ptr(i int32) *int32 { return &i }
