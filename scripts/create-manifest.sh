#!/bin/bash
set -e

# This script generates a single manifest file for installing the operator.
# This file is attached to each release with the appropriate container image for
# quick installation.

# Set the first argument as the operator container image tag. Default to "test".
OPERATOR_IMAGE="${1:-test}"

# List of manifests files to combine to form a single operator manifest file.
declare -a manifestfiles=(
    "deploy/crds/storageos_v1_storageoscluster_crd.yaml"
    "deploy/crds/storageos_v1_storageosupgrade_crd.yaml"
    "deploy/crds/storageos_v1_job_crd.yaml"
    "deploy/crds/storageos_v1_nfsserver_crd.yaml"
    "deploy/namespace.yaml"
    "deploy/role.yaml"
    "deploy/service_account.yaml"
    "deploy/role_binding.yaml"
)

# Path of the operator install manifest file.
INSTALL_MANIFEST="storageos-operator.yaml"

# Delete the existing manifest.
rm -f $INSTALL_MANIFEST

for i in "${manifestfiles[@]}"
do
    echo "---" >> $INSTALL_MANIFEST
    echo "Copying $i"
    cat $i >> $INSTALL_MANIFEST
done

# Write the operator manifest with the proper container image tag.
echo "---" >> $INSTALL_MANIFEST
echo "Copying deploy/operator.yaml with image $OPERATOR_IMAGE"
build/yq w deploy/operator.yaml spec.template.spec.containers[0].image $OPERATOR_IMAGE >> $INSTALL_MANIFEST
