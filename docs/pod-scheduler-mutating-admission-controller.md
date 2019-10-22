# Pod Scheduler Mutating Admission Controller

Cluster operator supports running a mutating admission controller to
automatically set the `schedulerName` attribute of a pod to StorageOS scheduler.

## Webhook Server

The admission controller webhook server runs within the operator, managed by the
controller manager. The webhook server must support TLS. A self-signed TLS
certificate is generated within the operator and used when running the webhook
server. During the webhook server bootstrap process, the operator also creates a
`MutatingWebhookConfiguration` and a `Service` resources. A client cert for the
generated cert is set in the `MutatingWebhookConfiguration`. This client cert is
used by k8s when it communicates with the webhook server. The `Service` is used
by the k8s to communicate with the webhook server. The `Service` is initialized
with label selectors to select the operator pod with the port that serves the
webhook server.

The `MutatingWebhookConfiguration` is not bound to any namespace.
```
$ kubectl get mutatingwebhookconfigurations
NAME                    CREATED AT
storageos-scheduler-webhook   2019-10-22T08:03:04Z
```

The `Service` is created in the namespace where the operator is deployed.
```
$ kubectl -n storageos-operator get service
NAME                    TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)   AGE
storageos-scheduler-webhook   ClusterIP   10.108.236.78   <none>        443/TCP   3m51s
```

If the operator pod restarts for some reason, the previous TLS certificate is
lost because it was stored within the pod. The operator generates and signs a
new cert and updates the `MutatingWebhookConfiguration` with a new client cert.
This may change in the future, the certificate may be generated once and stored
in a k8s `Secret` and shared among multiple instances of the operator, making
the cert persistent. If the certificate expires, the operator handles renewal of
the certificate automatically.

If the webhook server is not available due to some reason, the failure policy of
`MutatingWebhookConfiguration` is set to ignore the webhook.

## Mutation Conditions

The webhook server handler conditionally mutates the pods. The conditions for
mutating are:

- The pod has at least one volume that is managed by StorageOS.
- The StorageOS cluster has the scheduler enabled.

Both the above conditions must be true for the mutation to applied.

## Deployment

The admission controller is enabled in the default installation of the operator.
In order to disable the admission controller, the `Deployment` config of the
operator can be modified to set the environment variable `ENABLE_SCHEDULER` to
`"false"`.

## Skipping Mutation

If a pod uses StorageOS volume but wants to skip the mutation, this can be
achieved by adding an annotation to the pod:
```yaml
apiVersion: v1
kind: Pod
metadata:
    ...
    annotations:
        storageos.com/scheduler: "false"
    ...
```
The value must be a string.
