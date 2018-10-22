package k8sutil

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

// GetK8SVersion queries and returns kubernetes server version.
func GetK8SVersion(client kubernetes.Interface) (string, error) {
	info, err := client.Discovery().ServerVersion()
	if err != nil {
		return "", err
	}
	return info.String(), nil
}

// EventRecorder creates and returns an EventRecorder which could be used to
// broadcast events for k8s objects.
func EventRecorder(kubeClient kubernetes.Interface) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: kubeClient.CoreV1().Events(""),
		},
	)
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: "storageoscluster-operator"},
	)
	return recorder
}
