package storageos

import (
	"fmt"
	"time"

	monitoringv1 "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/storageos/cluster-operator/pkg/util/k8s"
	"github.com/storageos/cluster-operator/pkg/util/k8s/resource"
)

const (
	// APIManagerName is the name used for the api manager deployment.
	APIManagerName = "storageos-api-manager"
	// APIManagerMetricsName name is the name used for the api manager metrics service.
	APIManagerMetricsName = "storageos-api-manager-metrics"

	apiManagerMetricsPortName = "metrics"
	apiManagerMetricsPort     = int32(8080)
	createTimeout             = 5 * time.Second
	createPoll                = 100 * time.Millisecond
)

// createAPIManager deploys the StorageOS api-manager.
func (s *Deployment) createAPIManager() error {
	replicas := int32(2)
	if err := s.createServiceAccountForAPIManager(); err != nil {
		return err
	}
	if err := s.createClusterRoleForAPIManager(); err != nil {
		return err
	}
	if err := s.createClusterRoleBindingForAPIManager(); err != nil {
		return err
	}
	if err := s.createAPIManagerDeployment(replicas); err != nil {
		return err
	}
	return s.createAPIManagerMetrics()
}

// createAPIManagerDeployment creates a Deployment for api-manager with
// the given replicas.
func (s Deployment) createAPIManagerDeployment(replicas int32) error {
	// secretMode is set to readable by root, since api-manager runs as root
	// within the container.
	secretMode := int32(0600)
	podLabels := podLabelsForAPIManager(s.stos.Name)
	spec := &appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: &metav1.LabelSelector{
			MatchLabels: podLabels,
		},
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: podLabels,
			},
			Spec: corev1.PodSpec{
				ServiceAccountName: APIManagerSA,
				Containers: []corev1.Container{
					{
						Image:           s.stos.Spec.GetAPIManagerImage(),
						Name:            "api-manager",
						ImagePullPolicy: corev1.PullIfNotPresent,
						Command: []string{
							"/manager",
						},
						Args: []string{
							"--enable-leader-election",
							fmt.Sprintf("--metrics-addr=:%d", apiManagerMetricsPort),
						},
						Ports: []corev1.ContainerPort{
							{
								Name:          apiManagerMetricsPortName,
								ContainerPort: apiManagerMetricsPort,
							},
						},
						VolumeMounts: []corev1.VolumeMount{
							{
								Name:      "api-secret",
								MountPath: "/etc/storageos/secrets/api",
							},
						},
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "api-secret",
						VolumeSource: corev1.VolumeSource{
							Secret: &corev1.SecretVolumeSource{
								SecretName:  initSecretName,
								DefaultMode: &secretMode,
							},
						},
					},
				},
			},
		},
	}

	if err := s.addCommonPodProperties(&spec.Template.Spec); err != nil {
		return err
	}

	return s.k8sResourceManager.Deployment(APIManagerName, s.stos.Spec.GetResourceNS(), nil, spec).Create()
}

// deleteAPIManager deletes the API Manager deployment and all the associated
// resources.
func (s Deployment) deleteAPIManager() error {
	ns := s.stos.Spec.GetResourceNS()
	if err := s.k8sResourceManager.Service(APIManagerMetricsName, ns, nil, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.Deployment(APIManagerName, ns, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ClusterRoleBinding(APIManagerClusterBindingName, nil, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ClusterRole(APIManagerClusterRoleName, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ServiceAccount(APIManagerSA, ns, nil).Delete(); err != nil {
		return err
	}
	return nil
}

// podLabelsForAPIManager takes the name of a cluster custom resource and the
// kind of helper, and returns labels for the pods of the api-manager.
func podLabelsForAPIManager(name string) map[string]string {
	labels := map[string]string{
		"app":            appName,
		"storageos_cr":   name,
		k8s.AppComponent: APIManagerName,
	}
	return k8s.AddDefaultAppLabels(name, labels)
}

// createAPIManagerMetrics creates a service for the metrics endpoint and
// creates a ServiceMonitor if a Prometheus installation is detected.
func (s Deployment) createAPIManagerMetrics() error {
	ns := s.stos.Spec.GetResourceNS()
	svcSpec := &v1.ServiceSpec{
		Ports: []corev1.ServicePort{
			{
				Name: apiManagerMetricsPortName,
				Port: apiManagerMetricsPort,
			},
		},
		Selector: map[string]string{
			k8s.AppComponent: APIManagerName,
		},
	}

	labels := podLabelsForAPIManager(s.stos.Name)
	svcLabels := labels
	svcLabels[k8s.ServiceFor] = APIManagerName

	if err := s.k8sResourceManager.Service(APIManagerMetricsName, ns, svcLabels, nil, svcSpec).Create(); err != nil {
		return err
	}

	// Only create a ServiceMonitor if the CRD exists. The ServiceMonitor will be
	// deleted when the Service it refers to has been deleted.
	exists, err := k8sutil.ResourceExists(s.discoveryClient, resource.APIservicemonitorv1, resource.ServiceMonitorKind)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	// Wait for service creation.
	var svc *corev1.Service
	err = wait.Poll(createPoll, createTimeout, func() (bool, error) {
		svc, err = s.k8sResourceManager.Service(APIManagerMetricsName, ns, svcLabels, nil, svcSpec).Get()
		if client.IgnoreNotFound(err) != nil {
			return false, err
		}
		return true, nil
	})
	if err != nil {
		return err
	}

	var endpoints []monitoringv1.Endpoint
	for _, port := range svc.Spec.Ports {
		endpoints = append(endpoints, monitoringv1.Endpoint{Port: port.Name})
	}
	smSpec := &monitoringv1.ServiceMonitorSpec{
		Selector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				k8s.AppComponent: APIManagerName,
			},
		},
		Endpoints: endpoints,
	}

	// Create ServiceMonitor resources, but don't error if Prometheus not installed.
	return s.k8sResourceManager.ServiceMonitor(APIManagerMetricsName, ns, labels, nil, svc, smSpec).Create()
}
