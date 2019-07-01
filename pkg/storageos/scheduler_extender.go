package storageos

import (
	"bytes"
	"fmt"
	"text/template"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	policyConfigMapName          = "storageos-scheduler-policy"
	policyConfigKey              = "policy.cfg"
	schedulerConfigConfigMapName = "storageos-scheduler-config"
	schedulerConfigKey           = "config.yaml"

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
	schedulerDeployment := s.schedulerDeployment(replicas)
	return s.createOrUpdateObject(schedulerDeployment)
}

// schedulerDeployment returns a scheduler extender Deployment object. This
// contains the deployment configuration of the external kube-scheduler.
func (s Deployment) schedulerDeployment(replicas int32) *appsv1.Deployment {
	podLabels := podLabelsForScheduler(s.stos.Name)
	dep := &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      schedulerExtenderName,
			Namespace: s.stos.Spec.GetResourceNS(),
			Labels: map[string]string{
				"app": "storageos",
			},
		},
		Spec: appsv1.DeploymentSpec{
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
		},
	}

	// Add pod toleration for quick recovery on node failure.
	addPodTolerationForRecovery(&dep.Spec.Template.Spec)

	return dep
}

// schedulerContainers returns a list of containers that should be part of the
// scheduler extender deployment.
func (s Deployment) schedulerContainers() []corev1.Container {
	return []corev1.Container{
		{
			Image:           s.stos.Spec.GetHyperkubeImage(s.k8sVersion),
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
	if err := s.deleteObject(s.getDeploymentByName(schedulerExtenderName)); err != nil {
		return err
	}
	if err := s.deleteObject(s.getConfigMap(policyConfigMapName)); err != nil {
		return err
	}
	if err := s.deleteObject(s.getConfigMap(schedulerConfigConfigMapName)); err != nil {
		return err
	}
	if err := s.deleteClusterRoleBinding(SchedulerClusterBindingName); err != nil {
		return err
	}
	if err := s.deleteServiceAccount(SchedulerSA); err != nil {
		return err
	}
	if err := s.deleteClusterRole(SchedulerClusterRoleName); err != nil {
		return err
	}
	return nil
}

// getConfigMap returns an empty ConfigMap object. This can be used while
// creating a configmap resource.
func (s Deployment) getConfigMap(name string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: s.stos.Spec.GetResourceNS(),
		},
	}
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
	policyConfigMap := s.getConfigMap(policyConfigMapName)
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
	policyData := schedulerPolicyTemplate{
		FilterVerb:     "filter",
		PrioritizeVerb: "prioritize",
		EnableHTTPS:    false,
		URLPrefix:      fmt.Sprintf("http://%s:5705/v1/scheduler", serviceEndpoint),
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
	policyConfigMap.Data = map[string]string{
		"policy.cfg": policyConfig.String(),
	}
	return s.createOrUpdateObject(policyConfigMap)
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
	configConfigMap := s.getConfigMap(schedulerConfigConfigMapName)
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
		SchedulerName:  schedulerExtenderName,
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
	configConfigMap.Data = map[string]string{
		"config.yaml": schedConfig.String(),
	}
	return s.createOrUpdateObject(configConfigMap)
}

// podLabelsForScheduler returns labels for the scheduler pod.
func podLabelsForScheduler(name string) map[string]string {
	return map[string]string{
		"app":          appName,
		"storageos_cr": name,
		"kind":         deploymentKind,
	}
}
