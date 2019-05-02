#!/usr/bin/env bash

set -Eeuxo pipefail

# This script sets up openshift test environment on ubuntu trusty, the same
# environment that we create in the CI.

# Install kernel modules and enable LIO.
sudo apt -y install linux-modules-extra-$(uname -r)
sudo mount --make-shared /sys
sudo mount --make-shared /
sudo mount --make-shared /dev
docker run --name enable_lio --privileged --rm --cap-add=SYS_ADMIN -v /lib/modules:/lib/modules -v /sys:/sys:rshared storageos/init:0.1

# Download oc to spin up openshift on local docker instance
curl -Lo oc.tar.gz https://github.com/openshift/origin/releases/download/v3.11.0/openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit.tar.gz
# Put oc binary in path
tar xvzOf oc.tar.gz openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit/oc > oc && chmod +x oc && sudo mv oc /usr/local/bin/
# Install autocompletion
oc completion bash > oc
sudo mv oc /etc/bash_completion.d/
# Start oc cluster
oc cluster up
# Become cluster admin
oc login -u system:admin
# Install kubectl
curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.11.3/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
# Install autocompletion
kubectl completion bash > kubectl
sudo mv kubectl /etc/bash_completion.d/

# Install go
wget https://dl.google.com/go/go1.12.4.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.12.4.linux-amd64.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> ~/.profile
export PATH=$PATH:/usr/local/go/bin

# Install storageos-cli
curl -Lo storageos https://github.com/storageos/go-cli/releases/download/1.0.0/storageos_linux_amd64 && chmod +x storageos && sudo mv storageos /usr/local/bin/

# Install jq
sudo apt -y update && sudo apt install -y jq

# Install yq
curl -Lo yq https://github.com/mikefarah/yq/releases/download/2.3.0/yq_linux_amd64 && chmod +x yq && sudo mv yq /usr/local/bin/

# Get cluster-operator code
go get -v -d github.com/storageos/cluster-operator

# Install operator-sdk
cd ~/go/src/github.com/storageos/cluster-operator/
make install-operator-sdk
cd -
