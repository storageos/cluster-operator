# storageos-operator


Setup/Development:
1. Install [operator-sdk](https://github.com/operator-framework/operator-sdk#quick-start).
2. Run `operator-sdk generate k8s` if there's a change in api type.
3. Build operator container with `operator-sdk build storageos/storageoscluster-operator:<tag>`
4. Install the configs in `deploy/`
    - Apply `rbac.yaml` to grant all the permissions
    - Apply `crd.yaml` to define the custom resources.
    - Apply `operator.yaml` to install the operator. Change the container image in this file when installing a new operator.
    - Apply `cr.yaml` to create a custom resource.

**NOTE**: Installing StorageOS on minikube would fail because the init container fails to enable lio on the minikube host.

For development, run the operator outside of the k8s cluster by running:
```
$ OPERATOR_NAME=storageos-operator operator-sdk up local
```
Pass kubeconfig:
```
OPERATOR_NAME=storageos-operator operator-sdk up local --kubeconfig=/path/to/kubeconfig
```

After creating a resource, query the resource:
```
$ kubectl get storageos
NAME                READY     STATUS    AGE
example-storageos   3/3       Running   4m
```

Get all the details about the resource:
```
$ kubectl describe storageos/example-storageos
Name:         example-storageos
Namespace:    default
Labels:       <none>
Annotations:  kubectl.kubernetes.io/last-applied-configuration={"apiVersion":"node.storageos.com/v1alpha1","kind":"StorageOS","metadata":{"annotations":{},"name":"example-storageos","namespace":"default"},"spec":{"...
API Version:  node.storageos.com/v1alpha1
Kind:         StorageOS
Metadata:
  Creation Timestamp:  2018-07-21T12:57:11Z
  Generation:          1
  Resource Version:    10939030
  Self Link:           /apis/node.storageos.com/v1alpha1/namespaces/default/storageoses/example-storageos
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
