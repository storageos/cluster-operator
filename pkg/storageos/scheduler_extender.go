package storageos

import (
	"bytes"
	"fmt"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/storageos/cluster-operator/pkg/util/k8s"
)

const (
	policyConfigMapName          = "storageos-scheduler-policy"
	policyConfigKey              = "policy.cfg"
	schedulerConfigConfigMapName = "storageos-scheduler-config"
	schedulerConfigKey           = "config.yaml"

	uriPathV1 = "/v1/scheduler"
	uriPathV2 = "/v2/k8s/scheduler"

	// schedulerReplicas is the number of instances of kube-scheduler.
	schedulerReplicas = 1
)

func (s *Deployment) createSchedulerExtender() error {
	// Create configmap with scheduler configuration and policy.
	if err := s.createSchedulerPolicy(); err != nil {
		return err
	}
	if err := s.createSchedulerConfiguration(); err != nil {
		return err
	}

	// Create RBAC related resources.
	if err := s.createClusterRoleForScheduler(); err != nil {
		return err
	}
	if err := s.createServiceAccountForScheduler(); err != nil {
		return err
	}
	if err := s.createClusterRoleBindingForScheduler(); err != nil {
		return err
	}

	// Replicas of kube-scheduler deployment.
	replicas := int32(schedulerReplicas)

	// Create the deployment.
	return s.createSchedulerDeployment(replicas)
}

// createSchedulerDeployment returns a scheduler extender Deployment object. This
// contains the deployment configuration of the external kube-scheduler.
func (s Deployment) createSchedulerDeployment(replicas int32) error {
	podLabels := podLabelsForScheduler(s.stos.Name)
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
				ServiceAccountName: SchedulerSA,
				Containers:         s.schedulerContainers(),
				Volumes:            s.schedulerVolumes(),
			},
		},
	}

	// Add cluster config tolerations.
	s.addTolerations(&spec.Template.Spec)

	// Add pod toleration for quick recovery on node failure.
	addPodTolerationForRecovery(&spec.Template.Spec)

	return s.k8sResourceManager.Deployment(SchedulerExtenderName, s.stos.Spec.GetResourceNS(), nil, spec).Create()
}

// schedulerContainers returns a list of containers that should be part of the
// scheduler extender deployment.
func (s Deployment) schedulerContainers() []corev1.Container {
	return []corev1.Container{
		{
			Image:           s.stos.Spec.GetKubeSchedulerImage(s.k8sVersion),
			Name:            "storageos-scheduler",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Args: []string{
				"kube-scheduler",
				"--config=/storageos-scheduler/config.yaml",
				"-v=4",
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "storageos-scheduler-config",
					MountPath: "/storageos-scheduler",
				},
			},
		},
	}
}

// schedulerVolumes returns a list of volumes that should be part of the
// scheduler extender deployment.
func (s Deployment) schedulerVolumes() []corev1.Volume {
	return []corev1.Volume{
		{
			Name: schedulerConfigConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: schedulerConfigConfigMapName},
				},
			},
		},
	}
}

// deleteSchedulerExtender deletes all the scheduler related resources.
func (s Deployment) deleteSchedulerExtender() error {
	namespace := s.stos.Spec.GetResourceNS()
	if err := s.k8sResourceManager.Deployment(SchedulerExtenderName, namespace, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ConfigMap(policyConfigMapName, namespace, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ConfigMap(schedulerConfigConfigMapName, namespace, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ClusterRoleBinding(SchedulerClusterBindingName, nil, nil, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ServiceAccount(SchedulerSA, namespace, nil).Delete(); err != nil {
		return err
	}
	if err := s.k8sResourceManager.ClusterRole(SchedulerClusterRoleName, nil, nil).Delete(); err != nil {
		return err
	}
	return nil
}

// schedulerPolicyTemplate contains fields for rendering the scheduler policy.
type schedulerPolicyTemplate struct {
	FilterVerb     string
	PrioritizeVerb string
	EnableHTTPS    bool
	URLPrefix      string
}

// createSchedulerPolicy creates a configmap with kube-scheduler policy.
func (s Deployment) createSchedulerPolicy() error {
	policyConfigurationTemplate := `
    {
      "kind" : "Policy",
      "apiVersion" : "v1",
      "predicates" : [
        {"name" : "PodFitsHostPorts"},
        {"name" : "PodFitsResources"},
        {"name" : "NoDiskConflict"},
        {"name" : "MatchNodeSelector"},
        {"name" : "HostName"}
      ],
      "extenders" : [{
        "urlPrefix": "{{.URLPrefix}}",
        "filterVerb": "{{.FilterVerb}}",
        "prioritizeVerb": "{{.PrioritizeVerb}}",
        "weight": 1,
        "enableHttps": {{.EnableHTTPS}},
        "nodeCacheCapable": false
      }]
    }
`
	// Service address format: <service-name>.<namespace>.svc.cluster.local.
	serviceEndpoint := fmt.Sprintf("%s.%s.svc.cluster.local", s.stos.Spec.GetServiceName(), s.stos.Spec.GetResourceNS())
	uriPath := uriPathV1
	if s.nodev2 {
		uriPath = uriPathV2
	}
	policyData := schedulerPolicyTemplate{
		FilterVerb:     "filter",
		PrioritizeVerb: "prioritize",
		EnableHTTPS:    false,
		URLPrefix:      fmt.Sprintf("http://%s:5705%s", serviceEndpoint, uriPath),
	}

	// Render the policy configuration.
	var policyConfig bytes.Buffer
	tmpl, err := template.New("policyConfig").Parse(policyConfigurationTemplate)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(&policyConfig, policyData); err != nil {
		return err
	}

	// Add the policy configuration in the configmap.
	data := map[string]string{
		"policy.cfg": policyConfig.String(),
	}
	return s.k8sResourceManager.ConfigMap(policyConfigMapName, s.stos.Spec.GetResourceNS(), nil, data).Create()
}

// schedulerConfigTemplate contains fields for rendering the scheduler
// configuration.
type schedulerConfigTemplate struct {
	SchedulerName  string
	PolicyName     string
	Namespace      string
	LeaderElection bool
}

// createSchedulerConfiguration creates a configmap with kube-scheduler
// configuration.
func (s Deployment) createSchedulerConfiguration() error {
	configTemplate := `
    apiVersion: "kubescheduler.config.k8s.io/v1alpha1"
    kind: KubeSchedulerConfiguration
    schedulerName: {{.SchedulerName}}
    algorithmSource:
      policy:
        configMap:
          namespace: {{.Namespace}}
          name: {{.PolicyName}}
    leaderElection:
      leaderElect: {{.LeaderElection}}
      lockObjectName: {{.SchedulerName}}
      lockObjectNamespace: {{.Namespace}}
`
	schedConfigData := schedulerConfigTemplate{
		SchedulerName:  SchedulerExtenderName,
		PolicyName:     policyConfigMapName,
		Namespace:      s.stos.Spec.GetResourceNS(),
		LeaderElection: true,
	}

	// Render the scheduler configuration.
	var schedConfig bytes.Buffer
	tmpl, err := template.New("schedConfig").Parse(configTemplate)
	if err != nil {
		return err
	}
	if err := tmpl.Execute(&schedConfig, schedConfigData); err != nil {
		return err
	}

	// Add the configuration in the configmap.
	data := map[string]string{
		"config.yaml": schedConfig.String(),
	}
	return s.k8sResourceManager.ConfigMap(schedulerConfigConfigMapName, s.stos.Spec.GetResourceNS(), nil, data).Create()
}

// podLabelsForScheduler returns labels for the scheduler pod.
func podLabelsForScheduler(name string) map[string]string {
	// Combine CSI Helper specific labels with the default app labels.
	labels := map[string]string{
		"app":          appName,
		"storageos_cr": name,
		"kind":         deploymentKind,
	}
	return k8s.AddDefaultAppLabels(name, labels)
}
