# Pod Scheduler Mutating Admission Controller

The cluster operator works with the api-manager to create a mutating webhook
admission controller that configures Pods with StorageOS volumes to use the
StorageOS scheduler.  

The cluster operator creates the mutating webhook configuration, the service,
and the RBAC required by the api-manager.

The api-manager runs the webhook server and handles certificate signing and
rotation.  See
https://github.com/storageos/api-manager/tree/master/controllers/admission/scheduler
for more information.

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
