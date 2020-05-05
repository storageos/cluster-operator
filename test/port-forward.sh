#!/bin/sh

# This script is used to test the storageos node label update by
# node-controller.
# All the outputs of the commands are redirected to a file in order to print
# only the final result in the stdout, which is read by go test.

# Get the storageos daemonset pod name and create a port-forward to that pod.
# Port forward from host is not at 5705 because in the environments where the
# host runs k8s directly, 5705 will already be in use by the storageos running
# on the host. In case of KinD, the host port remains free. In order to test
# this on both the environment, the host port is changed.
podname=$(kubectl -n kube-system get pods -o=name | grep daemonset)
kubectl -n kube-system port-forward $podname 5709:5705 >> pf.log 2>&1 &
pid=$!

sleep 7

# Use storageos-cli to authenticate and get the node label.
# The main test adds a label to the k8s node before running this script.
# Check if that label is added to the storageos node as well.
export STORAGEOS_HOST=127.0.0.1:5709
storageos login $STORAGEOS_HOST \
    --username storageos \
    --password storageos \
    >> pf.log 2>&1
storageos node inspect | jq first.labels | grep -q '"foo10": "bar10"'
result=$?

# Kill the port-forward process.
kill $pid

echo $result
