#!/bin/bash
set -e

# This script generates a single manifest file for installing the operator.
# This file is attached to each release with the appropriate container image for
# quick installation.

# Set the first argument as the operator container image tag. Default to "test".
OPERATOR_IMAGE="${1:-test}"
YQ=build/yq3

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

# Get the number of env vars in operator container configuration.
envVarIndex=$($YQ r $OPERATOR_MANIFEST spec.template.spec.containers[0].env --length)

# Insert env vars at the end of the env var list in container config.
if [ -n "$JAEGER_ENDPOINT" ]; then
    $YQ w -i $OPERATOR_MANIFEST spec.template.spec.containers[0].env[$envVarIndex].name JAEGER_ENDPOINT
    $YQ w -i $OPERATOR_MANIFEST spec.template.spec.containers[0].env[$envVarIndex].value $JAEGER_ENDPOINT
fi
# Increment the env var index before inserting more env var.
envVarIndex=$((envVarIndex+1))

if [ -n "$JAEGER_SERVICE_NAME" ]; then
    $YQ w -i $OPERATOR_MANIFEST spec.template.spec.containers[0].env[$envVarIndex].name JAEGER_SERVICE_NAME
    $YQ w -i $OPERATOR_MANIFEST spec.template.spec.containers[0].env[$envVarIndex].value $JAEGER_SERVICE_NAME
fi

# Write the operator manifest with the proper container image tag.
echo "---" >> $INSTALL_MANIFEST
echo "Copying $OPERATOR_MANIFEST with image $OPERATOR_IMAGE"
$YQ w $OPERATOR_MANIFEST spec.template.spec.containers[0].image $OPERATOR_IMAGE >> $INSTALL_MANIFEST

if [ -f "$OPERATOR_MANIFEST" ]; then
    rm -f "$OPERATOR_MANIFEST"
fi

