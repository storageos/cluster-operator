# Mutating Webhook Configuration

The cluster operator works with the api-manager to create a mutating webhook
admission controller that modifies Pods and PVCs when they are created.  

The cluster operator creates the mutating webhook configuration, the service,
and the RBAC required by the api-manager.

The api-manager runs the webhook server and handles certificate signing and
rotation.  

See https://github.com/storageos/api-manager/blob/master/README.md for more
information.

## Deployment

The admission controller is enabled in the default installation of the operator.
