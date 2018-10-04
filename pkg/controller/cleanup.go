package controller

import (
	"context"
	"fmt"

	daemonsetv1beta1 "github.com/darkowlzz/daemonset-job/pkg/apis/daemonset/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func cleanup(client client.Client) error {
	job := &daemonsetv1beta1.Job{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "daemonset.darkowlzz.space/v1beta1",
			Kind:       "Job",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "clean-stos",
			Namespace: "default",
		},
		Spec: daemonsetv1beta1.JobSpec{
			Image:     "darkowlzz/cleanup:v0.0.2",
			Args:      []string{"/basetarget/storageos"},
			MountPath: "/var/lib",
		},
	}

	_ = job.DeepCopy()

	if err := client.Create(context.Background(), job); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create cleanup job: %v", err)
	}
	return nil
}
