#!/bin/bash
set -e

# Path to the local yq binary.
yq=build/yq

# This script reads all the configurations from OLM configmap, combines with
# *-changes.yaml file, if applicable, and updates the CSV, CRD and package
# files. This script is used to sync the source configmap configurations with
# the OLM manifest files.

# Extract CSV from configmap, update with community operator changes, removes
# OLM scorecard proxy container and write to the final CSV file.
$yq r deploy/storageos-operators.configmap.yaml \
    data.clusterServiceVersions | $yq r - [0] | \
    $yq w -s deploy/olm/community-changes.yaml - | \
    $yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[1]' \
    > deploy/olm/storageos/storageos.clusterserviceversion.yaml

# Extract CSV from configmap, update with rhel operator changes, removes OLM
# scorecard proxy container and write to the final CSV file.
# Also removes CSI v0 and NFS RELATED_IMAGE env vars as Red Hat certification
# validates that they are set, even though v2 does not use them.
$yq r deploy/storageos-operators.configmap.yaml \
    data.clusterServiceVersions | $yq r - [0] | \
    $yq w -s deploy/olm/rhel-changes.yaml - | \
    $yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[1]' | \
    $yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[0].env[12]' | \
    $yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[0].env[11]' | \
    $yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[0].env[10]' | \
    $yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[0].env[9]' \
    > deploy/olm/csv-rhel/storageos.clusterserviceversion.yaml


# Read metadata file configmap and update the CRD files.

# storageoscluster
$yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | $yq r - [0] | tee \
    deploy/crds/storageos.com_storageosclusters_crd.yaml \
    deploy/olm/storageos/storageoscluster.crd.yaml > /dev/null
# job
$yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | $yq r - [1] | tee \
    deploy/crds/storageos.com_jobs_crd.yaml \
    deploy/olm/storageos/storageosjob.crd.yaml > /dev/null
# upgrade
$yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | $yq r - [2] | tee \
    deploy/crds/storageos.com_storageosupgrades_crd.yaml \
    deploy/olm/storageos/storageosupgrade.crd.yaml > /dev/null
# nfsserver
$yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | $yq r - [3] | tee \
    deploy/crds/storageos.com_nfsservers_crd.yaml \
    deploy/olm/storageos/storageosnfsserver.crd.yaml > /dev/null

# Extract package from configmap, update and write to the final file.
$yq r deploy/storageos-operators.configmap.yaml \
    data.packages | $yq r - [0] | $yq w -s deploy/olm/community-package-changes.yaml - > \
    deploy/olm/storageos/storageos.package.yaml
$yq r deploy/storageos-operators.configmap.yaml \
    data.packages | $yq r - [0] | $yq w -s deploy/olm/rhel-package-changes.yaml - > \
    deploy/olm/csv-rhel/storageos.package.yaml
