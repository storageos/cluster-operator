#!/bin/bash
set -e

# This script generates a single manifest file for installing the operator.
# This file is attached to each release with the appropriate container image for
# quick installation.

# Set the first argument as the operator container image tag. Default to "test".
OPERATOR_IMAGE="${1:-test}"

# List of manifests files to combine to form a single operator manifest file.
declare -a manifestfiles=(
    "deploy/crds/storageos.com_storageosclusters_crd.yaml"
    "deploy/crds/storageos.com_storageosupgrades_crd.yaml"
    "deploy/crds/storageos.com_jobs_crd.yaml"
    "deploy/crds/storageos.com_nfsservers_crd.yaml"
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

# Set operator install env vars. Be careful of the ordering if they change!
OPERATOR_MANIFEST=deploy/operator-generated.yaml
cp deploy/operator.yaml $OPERATOR_MANIFEST

if [ -n "$JAEGER_ENDPOINT" ]; then
    build/yq w -i $OPERATOR_MANIFEST spec.template.spec.containers[0].env[19].value $JAEGER_ENDPOINT
fi
if [ -n "$JAEGER_SERVICE_NAME" ]; then
    build/yq w -i $OPERATOR_MANIFEST spec.template.spec.containers[0].env[20].value $JAEGER_SERVICE_NAME
fi

# Write the operator manifest with the proper container image tag.
echo "---" >> $INSTALL_MANIFEST
echo "Copying $OPERATOR_MANIFEST with image $OPERATOR_IMAGE"
build/yq w $OPERATOR_MANIFEST spec.template.spec.containers[0].image $OPERATOR_IMAGE >> $INSTALL_MANIFEST

if [ -f "$OPERATOR_MANIFEST" ]; then
    rm -f "$OPERATOR_MANIFEST"
fi

