# StorageOS cluster-operator

[![Build Status](https://travis-ci.com/storageos/cluster-operator.svg?branch=master)](https://travis-ci.com/storageos/cluster-operator)
[![CircleCI](https://circleci.com/gh/storageos/cluster-operator.svg?style=svg)](https://circleci.com/gh/storageos/cluster-operator)

The StorageOS Cluster Operator deploys and configures a StorageOS cluster on
Kubernetes.

For quick installation of the cluster operator, refer to the install section in
the [releases page](https://github.com/storageos/cluster-operator/releases).

## Pre-requisites

* Kubernetes 1.9+
* Kubernetes must be configured to allow (configured by default in 1.10+):
  * Privileged mode containers (enabled by default)
  * Feature gate: MountPropagation=true.  This can be done by appending
    `--feature-gates MountPropagation=true` to the kube-apiserver and kubelet
    services.

Refer to the [StorageOS prerequisites docs](https://docs.storageos.com/docs/prerequisites/overview)
for more information.

## Setup/Development

1. Build operator container image with `make dev-image`. Publish or
  copy this container image to an existing k8s cluster to make it available
  for use within the cluster.
2. Generate install manifest file with `make install-manifest`. This
will generate `storageos-operator.yaml`.
3. Install the operator `kubectl create -f storageos-operator.yaml`
4. Install a `StorageOSCluster` by creating a custom resource
`kubectl create -f deploy/crds/*_storageoscluster_cr.yaml`.

**NOTE**: Installing StorageOS on Minikube is not currently supported due to
missing [kernel prerequisites](https://docs.storageos.com/docs/prerequisites/systemconfiguration).
There are custom built [Kubernetes in Docker (KinD)](https://github.com/kubernetes-sigs/kind)
node image compatible with StorageOS available at https://hub.docker.com/r/storageos/kind-node.

For development, run the operator outside of the k8s cluster by running:

```bash
make local-run
```

Build operator container image:

```bash
make operator-image OPERATOR_IMAGE=storageos/cluster-operator:test
```

This builds all the components and copies the binaries into the same container.

For any changes related to Operator Lifecycle Manager(OLM), update
`deploy/storageos-operators.configmap.yaml` and run `make metadata-update` to
automatically update all the CRD, CSV and package files.

After creating a StorageOSCluster resource, query the resource:

```bash
$ kubectl get storageoscluster
NAME                       READY     STATUS    AGE
example-storageoscluster   3/3       Running   4m
```

## Inspect a StorageOSCluster Resource

Get all the details about the cluster:

```bash
$ kubectl describe storageoscluster/example-storageoscluster
Name:         example-storageoscluster
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"storageos.com/v1","kind":"StorageOSCluster","metadata":{"annotations":{},"name":"example-storageoscluster","namespace":"default"},"spec":{"...
API Version:  storageos.com/v1
Kind:         StorageOSCluster
Metadata:
  Creation Timestamp:  2018-07-21T12:57:11Z
  Generation:          1
  Resource Version:    10939030
  Self Link:           /apis/storageos.com/v1/namespaces/default/storageosclusters/example-storageoscluster
  UID:                 955b24a4-8ce5-11e8-956a-1866da35eee2
Spec:
  Join:  test07
Status:
  Node Health Status:
  ...
  ...
  Nodes:
    test09
    test08
    test07
  Phase:  Running
  Ready:  3/3
Events:   <none>
```

## StorageOSCluster Resource Configuration

Once the StorageOS operator is running, a StorageOS cluster can be deployed by
creating a Cluster Configuration. The parameters specified in the configuration
will define how StorageOS is deployed, the rest of the installation details are
handled by the operator.

The following tables lists the configurable spec
parameters of the StorageOSCluster custom resource and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`secretRefName` | Reference name of storageos secret |
`secretRefNamespace` | Namespace of storageos secret |
`namespace` | Namespace where storageos cluster resources are created | `storageos`
`images.nodeContainer` | StorageOS node container image | `storageos/node:v2.3.1`
`images.initContainer` | StorageOS init container image | `storageos/init:v2.1.0`
`images.apiManagerContainer` | StorageOS API Manager container image | `storageos/api-manager:v1.0.0`
`images.csiNodeDriverRegistrarContainer` | CSI Node Driver Registrar Container image | Varies depending on Kubernetes version
`images.csiClusterDriverRegistrarContainer` | CSI Cluster Driver Registrar Container image |  Varies depending on Kubernetes version
`images.csiExternalProvisionerContainer` | CSI External Provisioner Container image |  Varies depending on Kubernetes version
`images.csiExternalAttacherContainer` | CSI External Attacher Container image |  Varies depending on Kubernetes version
`images.csiExternalResizerContainer` | CSI External Resizer Container image |  Varies depending on Kubernetes version
`ìmages.csiLivenessProbeContainer` | CSI Liveness Probe Container Image |  Varies depending on Kubernetes version
`service.name` | Name of the Service used by the cluster | `storageos`
`service.type` | Type of the Service used by the cluster | `ClusterIP`
`service.externalPort` | External port of the Service used by the cluster | `5705`
`service.internalPort` | Internal port of the Service used by the cluster | `5705`
`service.annotations` | Annotations of the Service used by the cluster |
`ingress.enable` | Enable ingress for the cluster | `false`
`ingress.hostname` | Hostname to be used in cluster ingress | `storageos.local`
`ingress.tls` | Enable TLS for the ingress | `false`
`ingress.annotations` | Annotations of the ingress used by the cluster |
`sharedDir` | Path to be shared with kubelet container when deployed as a pod | `/var/lib/kubelet/plugins/kubernetes.io~storageos`
`kvBackend.address` | Comma-separated list of addresses of external key-value store. (`1.2.3.4:2379,2.3.4.5:2379`) |
`kvBackend.backend` | Name of the key-value store to use. Set to `etcd` for external key-value store. | `embedded`
`tlsEtcdSecretRefName` | Name of the secret object that contains the etcd TLS certs. |
`tlsEtcdSecretRefNamespace` | Namespace of the secret object that contains the etcd TLS certs. |
`pause` | Pause the operator for cluster maintenance | `false`
`debug` | Enable debug mode for all the cluster nodes | `false`
`disableFencing` | Disable Pod fencing | `false`
`disableTelemetry` | Disable telemetry reports | `false`
`disableTCMU` | Disable TCMU to allow co-existence with other storage systems but degrades performance | `false`
`forceTCMU` | Forces TCMU to be enabled or causes StorageOS to abort startup | `false`
`disableScheduler` | Disable StorageOS scheduler for data locality | `false`
`nodeSelectorTerms` | Set node selector for storageos pod placement, including NFS pods |
`tolerations` | Set pod tolerations for storageos pod placement |
`resources` | Set resource requirements for the containers |
`k8sDistro` | The name of the Kubernetes distribution is use, e.g. `rancher` or `eks` |
`storageClassName` | The name of the default StorageClass created for StorageOS volumes | `fast`

## Upgrading a StorageOS Cluster

An existing StorageOS cluster can be upgraded to a new version of StorageOS by
creating an Upgrade Configuration. The cluster-operator takes care of
downloading the new container image and updating all the nodes with new version
of StorageOS.
An example of `StorageOSUpgrade` resource is [storageos_v1_storageosupgrade_cr.yaml](/deploy/crds/storageos_v1_storageosupgrade_cr.yaml).

Only offline upgrade is supported for now by cluster-operator. During the
upgrade, StorageOS maintenance mode is enabled, the applications that use
StorageOS volumes are scaled down and the whole StorageOS cluster is restarted
with a new version. Once the StorageOS cluster becomes usable, the applications
are scaled up to their previous configuration. Once the update is complete, make
sure to delete the upgrade resource to put the StorageOS cluster in normal mode.
This will disable the maintenance mode.

Once an upgrade resource is created, events related to the upgrade can be
viewed in the upgrade object description. All the status and errors, if any,
encountered during the upgrade are posted as events.

```bash
$ kubectl describe storageosupgrades example-storageosupgrade
Name:         example-storageosupgrade
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"storageos.com/v1","kind":"StorageOSUpgrade","metadata":{"annotations":{},"name":"example-storageosupgrade","namespace":"default"},...
API Version:  storageos.com/v1
Kind:         StorageOSUpgrade
...
Spec:
  New Image:  storageos/node:1.0.0
Events:
  Type    Reason           Age   From                Message
  ----    ------           ----  ----                -------
  Normal  PullImage         4m    storageos-upgrader  Pulling the new container image
  Normal  PauseClusterCtrl  2m    storageos-upgrader  Pausing the cluster controller and enabling cluster maintenance mode
  Normal  UpgradeInit       2m    storageos-upgrader  StorageOS upgrade of cluster example-storageos started
  Normal  UpgradeComplete   0s    storageos-upgrader  StorageOS upgraded to storageos/node:1.0.0. Delete upgrade object to disable cluster maintenance mode
```

## StorageOSUpgrade Resource Configuration

The following table lists the configurable spec parameters of the
StorageOSUpgrade custom resource and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`newImage` | StorageOS node container image to upgrade to |

## Cleanup Old Configurations

StorageOS creates and saves its files at `/var/lib/storageos` on the hosts. This
also contains some configurations of the cluster. To do a fresh install of
StorageOS, these files need to be deleted.

__WARNING__: This will delete any existing data and won't be recoverable.

__NOTE__: When using an external etcd, the data related to storageos should also
be removed.

```bash
ETCDCTL_API=3 /usr/local/bin/etcdctl --endpoints http://storageos-etcd-server:2379 del --prefix storageos
```

The cluster-operator provides a `Job`resource that can execute certain tasks on
all nodes or on selected nodes. This can be used to easily perform cleanup
task. An example would be to create a `Job` resource:

```yaml
apiVersion: storageos.com/v1
kind: Job
metadata:
  name: cleanup-job
spec:
  image: darkowlzz/cleanup:v0.0.2
  args: ["/var/lib/storageos"]
  mountPath: "/var/lib"
  hostPath: "/var/lib"
  completionWord: "done"
  nodeSelectorTerms:
    - matchExpressions:
      - key: node-role.kubernetes.io/worker
        operator: In
        values:
        - "true"
```

When applied, this job will run `darkowlzz/cleanup` container on the nodes that
have label `node-role.kubernetes.io/worker` with value `"true"`, mounting
`/var/lib` and passing the argument `/var/lib/storageos`. This will run
`rm -rf /var/lib/storageos` in the selected nodes and cleanup all the storageos
files. To run it on all the nodes, remove the `nodeSelectorTerms` attribute.
On completion, the resource description shows that the task is completed and
can be deleted.

```bash
$ kubectl describe jobs.storageos.com cleanup-job
Name:         cleanup-job
Namespace:    default
...
...
Spec:
  Completion Word:
  Args:
    /var/lib/storageos
  Host Path:            /var/lib
  Image:                darkowlzz/cleanup:v0.0.2
  ...
Status:
  Completed:  true
Events:
  Type    Reason        Age   From                       Message
  ----    ------        ----  ----                       -------
  Normal  JobCompleted  39s   storageoscluster-operator  Job Completed. Safe to delete.
```

Deleting the resource, will terminate all the pods that were created to run the
task.

Internally, this `Job` is backed by a controller that creates pods using a
DaemonSet. Job containers have to be built in a specific way to achieve this
behavior.

In the above example, the cleanup container runs a shell script(`script.sh`):

```bash
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

And the container image is made with Dockerfile:

```dockerfile
FROM alpine:3.6
COPY script.sh .
RUN chmod u+x script.sh
ENTRYPOINT ["./script.sh"]
```

The script, after running the main command, enters into a sleep state, instead
of exiting. This is needed because we don't want the container to exit and start
again and again. Once completed, it echos "done". This is read by the Job
controller to figure out when the task is completed. Once all the pods have
completed the task, the Job status is completed and it can be deleted.

This can be extended to do other similar cluster management operations. This is
also used internally in the cluster upgrade process.

## Job (jobs.storageos.com) Resource Configuration

The following table lists the configurable spec parameters of the
Job custom resource and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`image` | Container image that the job runs |
`args` | Any arguments to be passed when the container is run |
`hostPath` | Path on the host that is mounted on the job container |
`mountPath` | Path on the job container where the hostPath is mounted |
`completionWord` | The word that job controller looks for in the pod logs to determine if the task is completed |
`labelSelector` | Labels that are added to the job pods and are used to select them. |
`nodeSelectorTerms` | This can be used to select the nodes where the job runs. |

## TLS Support

To enable TLS, ensure that an ingress controller is installed in the cluster.
Set `ingress.enable` and `ingress.tls` to `true`.
Store the TLS cert and key as part of the storageos secret as:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: "storageos-api"
...
...
data:
  # echo -n '<secret>' | base64
  ...
  ...
  # Add base64 encoded TLS cert and key.
  tls.crt:
  tls.key:
```

## CSI

StorageOS supports the [Container Storage Interface (CSI)](https://github.com/container-storage-interface/spec)
to communicate with Kubernetes.

CSI credentials are required for deploying StorageOS.
Specify the CSI credentials as part of the storageos secret object as:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: "storageos-api"
...
...
data:
  # echo -n '<secret>' | base64
  ...
  ...
  csiProvisionUsername:
  csiProvisionPassword:
  csiControllerPublishUsername:
  csiControllerPublishPassword:
  csiNodePublishUsername:
  csiNodePublishPassword:
  csiControllerExpandUsername:
  csiControllerExpandPassword:
```

