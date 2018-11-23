package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// JobSpec defines the desired state of Job
type JobSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// Image is the container image to run as the job.
	Image string `json:"image"`

	// Args is an array of strings passed as an argumen to the job container.
	Args []string `json:"args"`

	// MountPath is the path in the job container where a volume is mounted.
	MountPath string `json:"mountPath"`

	// HostPath is the path in the host that's mounted into a job container.
	HostPath string `json:"hostPath"`

	// CompletionWord is the word that's looked for in the pod logs to find out
	// if a DaemonSet Pod has completed its task.
	CompletionWord string `json:"completionWord"`

	// LabelSelector is the label selector for the job Pods.
	LabelSelector string `json:"labelSelector"`

	// NodeSelectorTerms is the set of placement of the job pods using node
	// affinity requiredDuringSchedulingIgnoredDuringExecution.
	NodeSelectorTerms []corev1.NodeSelectorTerm `json:"nodeSelectorTerms"`
}

// GetLabelSelector returns Job's pod label selector.
func (s JobSpec) GetLabelSelector() string {
	if len(s.LabelSelector) != 0 {
		return s.LabelSelector
	}
	return "daemonset-job=true"
}

// JobStatus defines the observed state of Job
type JobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file

	// Completed indicates the complete status of job.
	Completed bool `json:"completed"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Job is the Schema for the jobs API
// +k8s:openapi-gen=true
type Job struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   JobSpec   `json:"spec,omitempty"`
	Status JobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// JobList contains a list of Job
type JobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Job `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Job{}, &JobList{})
}
