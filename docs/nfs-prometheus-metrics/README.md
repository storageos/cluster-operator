# Setup Prometheus for NFS Server Metrics

Install Prometheus-operator by cloning the [repo](https://github.com/coreos/prometheus-operator),
and applying the manifests in `bundle.yaml` at the root of the repo. This will
install the operator in the default namespace with all the permissions needed by
the operator.
```
$ kubectl get all
NAME                                      READY   STATUS    RESTARTS   AGE
pod/prometheus-operator-6685db5c6-f948k   1/1     Running   0          3d1h

NAME                          TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)    AGE
service/kubernetes            ClusterIP   10.96.0.1    <none>        443/TCP    3d23h
service/prometheus-operator   ClusterIP   None         <none>        8080/TCP   3d1h

NAME                                  READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/prometheus-operator   1/1     1            1           3d1h

NAME                                            DESIRED   CURRENT   READY   AGE
replicaset.apps/prometheus-operator-6685db5c6   1         1         1       3d1h
```

The operator also installs all the CRDs it supports. Operator logs:
```
level=info ts=2019-11-11T11:19:28.157847755Z caller=operator.go:634 component=alertmanageroperator msg="CRD created" crd=Alertmanager
level=info ts=2019-11-11T11:19:28.277350268Z caller=operator.go:1863 component=prometheusoperator msg="CRD created" crd=Prometheus
level=info ts=2019-11-11T11:19:28.375474437Z caller=operator.go:1863 component=prometheusoperator msg="CRD created" crd=ServiceMonitor
level=info ts=2019-11-11T11:19:28.38946191Z caller=operator.go:1863 component=prometheusoperator msg="CRD created" crd=PodMonitor
level=info ts=2019-11-11T11:19:28.522474051Z caller=operator.go:1863 component=prometheusoperator msg="CRD created" crd=PrometheusRule
```

CRD deletion is not handled by the operator. When the operator is uninstalled,
the CRDs have to be deleted manually.

__NOTE__: StorageOS operator can be installed without installing
prometheus-operator. If ServiceMonitor CR is not defined, StorageOS operator is
still granted permission to create it if it's defined later. When NFS Server is
created, the StorageOS operator checks if ServiceMonitor is defined before
attempting to create a ServiceMonitor for NFS Server metrics endpoint.

Create a StorageOS cluster using StorageOSCluster CR and provision a RWX volume.
This will create a NFS Server pod that also exposes the NFS Server metrics in a
separate service for metrics only, in addition to a service for the nfs server.

```
$ kubectl get svc
NAME                                               TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
kubernetes                                         ClusterIP   10.96.0.1       <none>        443/TCP    6h42m
prometheus-operator                                ClusterIP   None            <none>        8080/TCP   6h23m
pvc-7a71680f-2b2f-4b7e-9a6c-ea011f934822           ClusterIP   10.104.20.71    <none>        2049/TCP   12m
pvc-7a71680f-2b2f-4b7e-9a6c-ea011f934822-metrics   ClusterIP   10.111.46.143   <none>        80/TCP     12m
```

In the above, `pvc-7a71680f-2b2f-4b7e-9a6c-ea011f934822-metrics` is the NFS
server metrics service.

If the `servicemonitors.monitoring.coreos.com` CRD is defined in the cluster,
while provisioning the NFS Server, a ServiceMonitor object will also be created
pointing to the metrics service of the NFS Server.

```
$ kubectl describe servicemonitors.monitoring.coreos.com pvc-5eaaf8f0-e1d7-4d87-ba3d-753dca279b88-metrics
Name:         pvc-5eaaf8f0-e1d7-4d87-ba3d-753dca279b88-metrics
Namespace:    default
Labels:       app=storageos
              app.kubernetes.io/component=metrics
              storageos.com/nocache=true
Annotations:  <none>
API Version:  monitoring.coreos.com/v1
Kind:         ServiceMonitor
Metadata:
  Creation Timestamp:  2019-11-11T10:06:54Z
  Generation:          1
  Owner References:
    API Version:           v1
    Block Owner Deletion:  true
    Controller:            true
    Kind:                  Service
    Name:                  pvc-5eaaf8f0-e1d7-4d87-ba3d-753dca279b88
    UID:                   08c780e2-0de9-43c9-9c29-89b88590e093
  Resource Version:        79597
  Self Link:               /apis/monitoring.coreos.com/v1/namespaces/default/servicemonitors/pvc-5eaaf8f0-e1d7-4d87-ba3d-753dca279b88
  UID:                     0ac997d4-4132-49ef-8125-86dbf37eb56c
Spec:
  Endpoints:
    Port:  nfs
    Port:  metrics
  Namespace Selector:
  Selector:
    Match Labels:
      App:                          storageos
      app.kubernetes.io/component:  metrics
      storageos.com/nocache:        true
Events:                       <none>
```

The ServiceMonitor inherits labels from the respective NFS Server metrics
service.

To collect the NFS Server metrics in prometheus, create a Prometheus instance
with `serviceMonitorSelector` matching labels `app=storageos` and 
`app.kubernetes.io/component=metrics`. The prometheus server needs permissions
to be able to collect the metrics from the metrics endpoints. Create a
ServiceAccount with all the necessary RBAC permission by applying
[prometheus-rbac.yaml](prometheus-rbac.yaml) and create a prometheus server
using the created service account by applying [prometheus-nfs.yaml](prometheus-nfs.yaml).

```
$ kubectl get pods
NAME                                         READY   STATUS    RESTARTS   AGE
prometheus-operator-6685db5c6-4tfbq          1/1     Running   0          142m
prometheus-prometheus-nfs-0                  3/3     Running   1          33s
pvc-527a4462-50a4-490e-90a7-d2dd1224cf63-0   1/1     Running   0          118m
```

In the above, `prometheus-prometheus-nfs-0` is the prometheus server pod.
Once created, port-forward to the prometheus server pod and view the collected
metrics in the prometheus dashboard at `localhost:8080`.

```
$ kubectl port-forward prometheus-prometheus-nfs-0 8080:9090
```

Details about all the StorageOS NFS Server metrics can be found [here](https://github.com/storageos/nfs/#prometheus-metrics).
