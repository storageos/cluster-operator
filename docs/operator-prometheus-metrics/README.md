# Setup Prometheus for Operator Metrics

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

__NOTE__: Prometheus-operator must be installed before installing 
storageos-operator because the ServiceMonitor is created at startup. If
storageos-operator is already installed, restarting the operator pod by deleting
it should also result in startup and creation of Prometheus ServiceMonitor.

Install StorageOS Operator. It will create a Service for metrics and a
Prometheus ServiceMonitor.
```
$ kubectl -n storageos-operator get all
NAME                                             READY   STATUS    RESTARTS   AGE
pod/storageos-cluster-operator-77bffc958-wjpgx   1/1     Running   0          28m

NAME                                         TYPE        CLUSTER-IP       EXTERNAL-IP   PORT(S)             AGE
service/storageos-cluster-operator-metrics   ClusterIP   10.110.236.21    <none>        8383/TCP,8686/TCP   28m
service/storageos-webhook                    ClusterIP   10.103.159.179   <none>        443/TCP             28m

NAME                                         READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/storageos-cluster-operator   1/1     1            1           28m

NAME                                                   DESIRED   CURRENT   READY   AGE
replicaset.apps/storageos-cluster-operator-77bffc958   1         1         1       28m


$ kubectl -n storageos-operator get servicemonitors.monitoring.coreos.com 
NAME                                 AGE
storageos-cluster-operator-metrics   76s
```

In the above, `service/storageos-cluster-operator-metrics` is the metrics
service and servicemonitor `storageos-cluster-operator-metrics` is the
servicemonitor associated with the operator metrics service.

Create a Prometheus server instance with `serviceMonitorSelector` matching
label `name=storageos-cluster-operator`. Create ServiceMonitor for Prometheus
instance with all the RBAC permission by applying
[prometheus-rbac.yaml](prometheus-rbac.yaml) and create a prometheus server
using the created ServiceAccount by applying
[prometheus-stos-operator.yaml](prometheus-stos-operator.yaml).

```
$ kubectl -n storageos-operator get pods
NAME                                         READY   STATUS    RESTARTS   AGE
prometheus-prometheus-storageos-operator-0   3/3     Running   1          6s
storageos-cluster-operator-77bffc958-wjpgx   1/1     Running   0          8m44s
```

In the above, `prometheus-prometheus-storageos-operator-0` is the prometheus
server pod. Once created, port-forward to the prometheus server pod and view the
collected metrics in the prometheus dashboard at `localhost:8080`.

```
$ kubectl -n storageos-operator port-forward prometheus-prometheus-storageos-operator-0 8080:9090
```
