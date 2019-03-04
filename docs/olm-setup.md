# OLM Setup

To setup OLM in an existing cluster:
1. `$ source deploy/olm/olm.sh`
2. Install OLM by running: `olm_quick_install`. This will install OLM in `olm`
namespace.

`olm_quick_install` command clones the olm repository in `test/olm` directory. Once OLM is installed, the OLM console can be started by running the script
```
$ ./test/olm/scripts/run_console_local.sh
```
This starts a web UI for OLM at http://localhost:9000.


## Installing an Operator

To install an operator using OLM, a catalog source has to be installed first.
A catalog source is a collection of one or more operators, containing details
about how to install an operator and any prerequisites that should be met before
installing the operator. The prerequisites could be RBAC rules, CRDs or some
other operator.

Catalog sources are of two types:
1. grpc
2. internal

The gRPC catalogs are distributes in the form of container images, which expose a
gRPC socket when deployed. The catalog-operator, part of OLM, communicates with
the catalog source over gRPC.

The internal catalogs are in the form of a k8s configmap. The config map
contains the details about the operators.

[storageos-operators.configmap.yaml](/deploy/storageos-operators.configmap.yaml)
is an example of a configmap catalog source. Before creating a CatalogSource
object, this configmap must be created.
```
$ kubectl create -f deploy/storageos-operators.configmap.yaml
```

The above configmap name is `storageos-operators`. Create a CatalogSource object
with configmap name `storageos-operators`. An example of CatalogSource is in
[storageos-operators.catalogsource.yaml](/deploy/storageos-operators.catalogsource.yaml).
```
$ kubectl create -f deploy/storageos-operators.catalogsource.yaml
```

To initiate the installation of an operator, a Subscription of an operator has
to be created. An example of a Subscription is in
[storageos-operators.subscription.yaml](/deploy/storageos-operators.subscription.yaml).
```
$ kubectl create -f deploy/storageos-operators.subscription.yaml
```

Description of the subscription object shows the status of the installation: 
```
$ kubectl -n olm describe subscriptions storageos
Name:         storageos
Namespace:    olm
Labels:       <none>
Annotations:  <none>
API Version:  operators.coreos.com/v1alpha1
Kind:         Subscription
Metadata:
  Creation Timestamp:  2019-03-04T15:31:41Z
  Generation:          1
  Resource Version:    7487
  Self Link:           /apis/operators.coreos.com/v1alpha1/namespaces/olm/subscriptions/storageos
  UID:                 9bd2c739-3e92-11e9-85fa-02427239d77c
Spec:
  Channel:           alpha
  Name:              storageos
  Source:            storageos-catalog
  Source Namespace:  olm
Status:
  Current CSV:    storageosoperator.0.0.0
  Installed CSV:  storageosoperator.0.0.0
  Installplan:
    API Version:  operators.coreos.com/v1alpha1
    Kind:         InstallPlan
    Name:         install-storageosoperator.0.0.0-4hznd
    Uuid:         9bd68944-3e92-11e9-85fa-02427239d77c
  Last Updated:   2019-03-04T15:31:45Z
  State:          AtLatestKnown
Events:           <none>
```

If the installation fails, or remains in a pending state, the description would
show the reason for the failure or the pending state.

Once the operator installs successfully, it shows up in the OLM web console
under Catalog > Installed Operators.

To install storageos cluster, go to Catalog > Developer Catalog and create a
StorageOS Cluster resource.

__NOTE__: All the above operator installation can be done by running the command
`install_storageos_operator` after sourcing deploy/olm/olm.sh, but the default
catalog configmap in [storageos-operators.configmap.yaml](/deploy/storageos-operators.configmap.yaml)
refers to `storageos/cluster-operator:test` as the operator image. Change the
operator image to a valid image. `storageos/cluster-operator:test` is used in
the e2e test setup and is created in the test machine.

Similarly, to install storageos run `install_storageos`. This will create a
default secret and create a StorageOS Cluster object.

Some more helper commands:
- `uninstall_storageos` - removes the storageos installation installed through
the `install_storageos` command.
- `uninstall_storageos_operator` - removes the storageos operator installation
installed through the `install_storageos_operator` command.
- `uninstall_olm_quick` - removes the OLM setup from the cluster installed
through the `olm_quick_install` command.
