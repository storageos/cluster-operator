package util

import (
	goctx "context"
	"fmt"
	"strconv"
	"strings"

	framework "github.com/operator-framework/operator-sdk/pkg/test"
	"github.com/pkg/errors"
	deploy "github.com/storageos/cluster-operator/pkg/storageos"
	"github.com/storageos/cluster-operator/pkg/util/k8s"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetAPIManagerLogs() (string, error) {
	podList, err := PodListForSelector("kube-system", labels.SelectorFromSet(labels.Set{
		k8s.AppComponent: deploy.APIManagerName,
	}))
	if err != nil {
		return "", errors.Wrap(err, "failed to get api-manager pods")
	}

	output := ""
	for _, pod := range podList.Items {
		logs, err := GetPodLogs(pod.Name, pod.Namespace)
		if err != nil {
			return "", errors.Wrap(err, "failed to get api-manager pod")
		}
		for k, v := range logs {
			output = strings.Join([]string{output, v}, fmt.Sprintf("\n---- %s/%s\n", pod.Name, k))
		}
	}
	return output, nil
}

// PodListForSelector returns a list of pods that match the selector.
func PodListForSelector(namespace string, selector labels.Selector) (*corev1.PodList, error) {
	f := framework.Global

	options := metav1.ListOptions{LabelSelector: selector.String()}
	return f.KubeClient.CoreV1().Pods(namespace).List(goctx.TODO(), options)
}

func GetPodLogs(name string, namespace string) (map[string]string, error) {
	f := framework.Global
	key := client.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	pod := corev1.Pod{}
	if err := f.Client.Get(goctx.TODO(), key, &pod); err != nil {
		return nil, errors.Wrap(err, "failed to get pod")
	}

	logs := map[string]string{}
	for _, c := range pod.Spec.Containers {
		l, err := GetContainerLogs(pod.Name, pod.Namespace, c.Name, false)
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("failed to get container %s logs", c.Name))
		}

		logs[c.Name] = l
	}

	return logs, nil
}

func GetContainerLogs(podName string, namespace string, containerName string, previous bool) (string, error) {
	f := framework.Global
	logs, err := f.KubeClient.CoreV1().RESTClient().Get().
		Resource("pods").
		Namespace(namespace).
		Name(podName).SubResource("log").
		Param("container", containerName).
		Param("previous", strconv.FormatBool(previous)).
		Do(goctx.TODO()).
		Raw()
	if err != nil {
		return "", err
	}
	if err == nil && strings.Contains(string(logs), "Internal Error") {
		return "", fmt.Errorf("Fetched log contains \"Internal Error\": %q", string(logs))
	}
	return string(logs), err
}
