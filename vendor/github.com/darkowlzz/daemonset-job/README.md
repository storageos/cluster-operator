# daemonset-job

[![CircleCI](https://circleci.com/gh/darkowlzz/daemonset-job.svg?style=svg)](https://circleci.com/gh/darkowlzz/daemonset-job)

Kubernetes Operator for creating Jobs backed by Daemonsets.

Inspired by [Run Once DaemonSet on Kubernetes](https://blog.phymata.com/2017/05/13/run-once-daemonset-on-kubernetes/)
by Everett Toews.

Following is an example of a cleanup job using a container that runs
`rm -rf`. Filepath argument and mount path can be passed as part of the custom
resource Job.

```yaml
apiVersion: daemonset.darkowlzz.space/v1beta1
kind: Job
metadata:
  name: job-sample
spec:
  image: darkowlzz/cleanup:v0.0.2
  args: ["/basetarget/foo"]
  mountPath: "/tmp"
```

When deployed, this would execute `rm -rf /tmp/foo` on all the nodes.


## Components

### 1. DaemonSet Job Controller

This controller watches the DaemonSet Job CRD resources. When a Job object is
created, the controller creates and deploys a DaemonSet with container image
`JobSpec.Image`.
The controller then maintains or updates the DaemonSet, as per the `JobSpec`
attributes.

### 2. Sidecar Job Terminator Container

This sidecar container monitors pods that are created by the Job DaemonSet and
checks their status. When all the pods have completed their task, it terminates
the associated Job.

Source: https://github.com/darkowlzz/daemonset-job-terminator


## Special Container

DaemonSet Job depends on container logs to determine if a pod has completed its
task. Hence, the containers need to be made in a specific way.

An example of a Dockerfile for a container:

```dockerfile
FROM alpine:3.6
COPY script.sh .
RUN chmod u+x script.sh
ENTRYPOINT ["./script.sh"]
```

An example of the script above:

```sh
#!/bin/ash

set -euo pipefail

# Gracefully handle the TERM signal sent when deleting the daemonset
trap 'exit' TERM

# This is the main command that's run by this script on
# all the nodes.
rm -rf $1

# Let the monitoring script know we're done.
echo "done"

# this is a workaround to prevent the container from exiting
# and k8s restarting the daemonset pod
while true; do sleep 1; done
```

Once the task is completed, the container logs `"done"` and goes into a
sleep mode. This is needed because DaemonSet requires pods to be running
all the time.

When the sidecar container finds the termination word in the logs of a
container, it counts the associated pod as completed. Once all the pods created
by the daemonset have completed their task, the sidecar container deletes the
parent Job custom resource. This results in cleanup of the daemonset and pods by
garbage collection.


## Deployment

Register the DaemonSet Job CRD:
```
$ kubectl apply -f config/crds/daemonset_v1beta1_job.yaml
```

Deploy the controller and sidecar:
```
$ kubectl apply -f deploy/
```

## Try it

The Job manifest at the top would run `rm -rf /tmp/foo` on all the nodes.
In a k8s cluster, after deploying the controller, create `/tmp/foo` file on all
the nodes.
Applying the Job manifest would delete the file and cleanup all the created
resources.
