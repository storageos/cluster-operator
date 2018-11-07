package k8sutil

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/storageos/cluster-operator/pkg/util/task"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

const (
	pvcStorageClassKey       = "volume.beta.kubernetes.io/storage-class"
	storageosProvisioner     = "storageos"
	replicaKey               = "stos/replicas-before-scale-down"
	deploymentUpdateTimeout  = 5 * time.Minute
	statefulSetUpdateTimeout = 10 * time.Minute
	defaultRetryInterval     = 10 * time.Second
)

// K8SOps is a kubernetes operations type which can be used to query and perform
// actions on a kubernetes cluster.
type K8SOps struct {
	client kubernetes.Interface
}

// NewK8SOps creates and returns a new k8sOps object.
func NewK8SOps(client kubernetes.Interface) *K8SOps {
	return &K8SOps{client: client}
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
				continue
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
		log.Printf("scaling down deployment: [%s] %s", d.GetNamespace(), d.GetName())

		t := func() (interface{}, bool, error) {
			dCopy, err := k.client.AppsV1().Deployments(d.GetNamespace()).Get(d.GetName(), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil, false, nil
				}

				return nil, true, err
			}

			if *dCopy.Spec.Replicas == 0 {
				log.Printf("app [%s] %s is already scaled down to 0", dCopy.Namespace, dCopy.Name)
				return nil, false, nil
			}

			dCopy.Annotations[replicaKey] = fmt.Sprintf("%d", *dCopy.Spec.Replicas)
			dCopy.Spec.Replicas = &valZero
			_, updateErr := k.client.AppsV1().Deployments(d.GetNamespace()).Update(dCopy)
			if updateErr != nil {
				log.Printf("failed to update Deployment %s: %v", dCopy.GetName(), updateErr)
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, deploymentUpdateTimeout, defaultRetryInterval); err != nil {
			return err
		}
	}

	for _, s := range ss.Items {
		log.Printf("scaling down statefulset: [%s] %s", s.GetNamespace(), s.GetName())

		t := func() (interface{}, bool, error) {
			sCopy, err := k.client.AppsV1().StatefulSets(s.GetNamespace()).Get(s.GetName(), metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return nil, false, nil
				}

				return nil, true, err
			}

			if *sCopy.Spec.Replicas == 0 {
				log.Printf("app [%s] %s is already scaled down to 0", sCopy.Namespace, sCopy.Name)
				return nil, false, nil
			}

			sCopy.Annotations[replicaKey] = fmt.Sprintf("%d", *sCopy.Spec.Replicas)
			sCopy.Spec.Replicas = &valZero
			_, updateErr := k.client.AppsV1().StatefulSets(s.GetNamespace()).Get(s.GetName(), metav1.GetOptions{})
			if updateErr != nil {
				log.Printf("failed to update StatefulSet %s: %v", sCopy.GetName(), updateErr)
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, statefulSetUpdateTimeout, defaultRetryInterval); err != nil {
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
		log.Printf("restoring app: [%s] %s", d.Namespace, d.Name)

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
				log.Printf("not restoring app: [%s] %s as no annotation found to track replica count", dCopy.GetNamespace(), dCopy.GetName())
				return nil, false, nil
			}

			i, err := strconv.Atoi(val)
			if err != nil {
				log.Printf("failed to read replica for %s: %v", dCopy.GetName(), err)
				return nil, false, nil
			}

			delete(dCopy.Annotations, replicaKey)
			dCopy.Spec.Replicas = int32Ptr(int32(i))
			_, updateErr := k.client.AppsV1().Deployments(dCopy.GetNamespace()).Update(dCopy)
			if updateErr != nil {
				log.Printf("failed to update Daemonset %s: %v", dCopy.GetName(), updateErr)
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, deploymentUpdateTimeout, defaultRetryInterval); err != nil {
			return err
		}
	}

	for _, s := range ss.Items {
		log.Printf("restoring app: [%s] %s", s.Namespace, s.Name)

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
				log.Printf("not restoring app: [%s] %s as no annotation found to track replica count", sCopy.GetNamespace(), sCopy.GetName())
				return nil, false, nil
			}

			i, err := strconv.Atoi(val)
			if err != nil {
				log.Printf("failed to read replica for %s: %v", sCopy.GetName(), err)
				return nil, false, nil
			}

			delete(sCopy.Annotations, replicaKey)
			sCopy.Spec.Replicas = int32Ptr(int32(i))
			_, updateErr := k.client.AppsV1().StatefulSets(sCopy.GetNamespace()).Update(sCopy)
			if updateErr != nil {
				log.Printf("failed to update StatefulSet %s: %v", sCopy.GetName(), updateErr)
				return nil, true, updateErr
			}

			return nil, false, nil
		}

		if _, err := task.DoRetryWithTimeout(t, statefulSetUpdateTimeout, defaultRetryInterval); err != nil {
			return err
		}

	}

	return nil
}

func int32Ptr(i int32) *int32 { return &i }
