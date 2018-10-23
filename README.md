# storageos cluster-operator


Setup/Development:
1. Install [operator-sdk](https://github.com/operator-framework/operator-sdk#quick-start).
2. Run `operator-sdk generate k8s` if there's a change in api type.
3. Build operator container with `operator-sdk build storageos/cluster-operator:<tag>`
4. Install the configs in `deploy/`
    - Apply `rbac.yaml` to grant all the permissions
    - Apply `crd.yaml` to define the custom resources.
    - Apply `operator.yaml` to install the operator. Change the container image in this file when installing a new operator.
    - Apply `cr.yaml` to create a custom resource.

**NOTE**: Installing StorageOS on minikube would fail because the init container fails to enable lio on the minikube host.

For development, run the operator outside of the k8s cluster by running:
```
$ OPERATOR_NAME=cluster-operator operator-sdk up local
```
Pass kubeconfig:
```
OPERATOR_NAME=cluster-operator operator-sdk up local --kubeconfig=/path/to/kubeconfig
```

After creating a resource, query the resource:
```
$ kubectl get storageoscluster
NAME                READY     STATUS    AGE
example-storageos   3/3       Running   4m
```

Get all the details about the resource:
```
$ kubectl describe storageoscluster/example-storageos
Name:         example-storageos
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"storageos.com/v1alpha1","kind":"StorageOSCluster","metadata":{"annotations":{},"name":"example-storageos","namespace":"default"},"spec":{"...
API Version:  storageos.com/v1alpha1
Kind:         StorageOSCluster
Metadata:
  Creation Timestamp:  2018-07-21T12:57:11Z
  Generation:          1
  Resource Version:    10939030
  Self Link:           /apis/storageos.com/v1alpha1/namespaces/default/storageosclusters/example-storageos
  UID:                 955b24a4-8ce5-11e8-956a-1866da35eee2
Spec:
  Join:  test07
Status:
  Node Health Status:
    Test 07:
      Directfs Initiator:  alive
      Director:            alive
      Kv:                  alive
      Kv Write:            alive
      Nats:                alive
      Presentation:        alive
      Rdb:                 alive
      Scheduler:           alive
    Test 08:
      Directfs Initiator:  alive
      Director:            alive
      Kv:                  alive
      Kv Write:            alive
      Nats:                alive
      Presentation:        alive
      Rdb:                 alive
      Scheduler:           alive
    Test 09:
      Directfs Initiator:  alive
      Director:            alive
      Kv:                  alive
      Kv Write:            alive
      Nats:                alive
      Presentation:        alive
      Rdb:                 alive
      Scheduler:           alive
  Nodes:
    test09
    test08
    test07
  Phase:  Running
  Ready:  3/3
Events:   <none>
```

# Configurations

The following tables lists the configurable spec parameters of the StorageOSCluster custom resource and their default values.

Parameter | Description | Default
--------- | ----------- | -------
`secretRefName` | Reference name of storageos secret |
`secretRefNamespace` | Namespace of storageos secret |
`namespace` | Namespace where storageos cluster resources are created | `storageos`
`images.nodeContainer` | StorageOS node container image | `storageos/node:1.0.0-rc4`
`images.initContainer` | StorageOS init container image | `storageos/init:0.1`
`images.csiDriverRegistrarContainer` | CSI Driver Registrar Container image | `quay.io/k8scsi/driver-registrar:v0.2.0`
`images.csiExternalProvisionerContainer` | CSI External Provisioner Container image | `quay.io/k8scsi/csi-provisioner:v0.3.0`
`images.csiExternalAttacherContainer` | CSI External Attacher Container image | `quay.io/k8scsi/csi-attacher:v0.3.0`
`csi.enable` | Enable CSI setup | `false`
`csi.enableProvisionCreds` | Enable CSI provision credentials | `false`
`csi.enableControllerPublishCreds` | Enable CSI controller publish credentials | `false`
`csi.enableNodePublishCreds` | Enable CSI node publish credentials | `false`
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

## CSI Credentials

To enable CSI Credentials, ensure that CSI is enabled by setting `csi.enable` to
`true`. Based on the type of credentials to enable, set the csi fields to `true`:
```yaml
apiVersion: "storageos.com/v1alpha1"
kind: "StorageOSCluster"
metadata:
  name: "example-storageos"
  namespace: "default"
spec:
  ...
  ...
  csi:
    enable: true
    enableProvisionCreds: true
    enableControllerPublishCreds: true
    enableNodePublishCreds: true
  ...
```

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
```
