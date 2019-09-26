#!/bin/bash
set -e

# This script generates metadata files and compares them with the checked-in
# metadata files. Any difference in the generated matadata files and checked-in
# files results in failure.
# Whenever a new custom resource is added, a diff checker for that CR must be
# added here.

TMP_DIR="/tmp"
OLM_CONFIGMAP_FILE="deploy/storageos-operators.configmap.yaml"

# check_diff takes two files and checks if they have any diff.
check_diff () {
    echo "checking diff: $@"
    if diff $1 $2; then
        echo "clean diff"
    else
        echo "bad diff"
        exit 1
    fi
}

echo "Checking if all the files are up-to-date..."

##################
# Check CSV files.
##################

# Check community operator CSV file.
csvfile=$TMP_DIR/community-csv.yaml
targetfile=deploy/olm/storageos/storageos.clusterserviceversion.yaml
# Generate a community CSV file.
yq r deploy/storageos-operators.configmap.yaml \
    data.clusterServiceVersions | yq r - [0] | \
    yq w -s deploy/olm/community-changes.yaml - | \
    yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[1]' \
    > $csvfile
check_diff $targetfile $csvfile

# Check rhel CSV file.
csvfile=$TMP_DIR/rhel-csv.yaml
targetfile=deploy/olm/csv-rhel/storageos.clusterserviceversion.yaml
# Generate a rhel CSV file.
yq r deploy/storageos-operators.configmap.yaml \
    data.clusterServiceVersions | yq r - [0] | \
    yq w -s deploy/olm/rhel-changes.yaml - | \
    yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[1]' \
    > $csvfile
check_diff $targetfile $csvfile


##################
# Check CRD files.
##################

# Check storageoscluster CRD files.
clusterfile=$TMP_DIR/storageoscluster.crd.yaml
targetfiles=(
    deploy/crds/storageos_v1_storageoscluster_crd.yaml
    deploy/olm/storageos/storageoscluster.crd.yaml
)
# Generate a storageoscluster CRD file.
yq r $OLM_CONFIGMAP_FILE \
    data.customResourceDefinitions | yq r - [0] > $clusterfile
# Compare all the files.
for f in "${targetfiles[@]}"
do
    check_diff $f $clusterfile
done

# Check job CRD files.
jobfile=$TMP_DIR/job.crd.yaml
targetfiles=(
    deploy/crds/storageos_v1_job_crd.yaml
    deploy/olm/storageos/storageosjob.crd.yaml
)
yq r $OLM_CONFIGMAP_FILE \
    data.customResourceDefinitions | yq r - [1] > $jobfile
for f in "${targetfiles[@]}"
do
    check_diff $f $jobfile
done

# Check upgrade CRD files.
upgradefile=$TMP_DIR/storageosupgrade.crd.yaml
targetfiles=(
    deploy/crds/storageos_v1_storageosupgrade_crd.yaml
    deploy/olm/storageos/storageosupgrade.crd.yaml
)
yq r $OLM_CONFIGMAP_FILE \
    data.customResourceDefinitions | yq r - [2] > $upgradefile
for f in "${targetfiles[@]}"
do
    check_diff $f $upgradefile
done

# Check nfsserver CRD file.
nfsfile=$TMP_DIR/nfsserver.crd.yaml
targetfiles=(
    deploy/crds/storageos_v1_nfsserver_crd.yaml
    deploy/olm/storageos/storageosnfsserver.crd.yaml
)
yq r $OLM_CONFIGMAP_FILE \
    data.customResourceDefinitions | yq r - [3] > $nfsfile
for f in "${targetfiles[@]}"
do
    check_diff $f $nfsfile
done


#####################
# Check package file.
#####################

# Extract package from configmap into a temporary file.
packagefile=$TMP_DIR/package.yaml
targetfile=deploy/olm/storageos/storageos.package.yaml
yq r deploy/storageos-operators.configmap.yaml \
    data.packages | yq r - [0] | \
    yq w -s deploy/olm/package-changes.yaml - > $packagefile
check_diff $targetfile $packagefile
