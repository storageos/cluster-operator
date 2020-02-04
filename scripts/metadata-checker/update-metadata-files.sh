#!/bin/bash
set -ex

# This script reads all the configurations from OLM configmap, combines with
# *-changes.yaml file, if applicable, and updates the CSV, CRD and package
# files. This script is used to sync the source configmap configurations with
# the OLM manifest files.

# Extract CSV from configmap, update with community operator changes, removes
# OLM scorecard proxy container and write to the final CSV file.
yq r deploy/storageos-operators.configmap.yaml \
    data.clusterServiceVersions | yq r - [0] | \
    yq w -s deploy/olm/community-changes.yaml - | \
    yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[1]' \
    > deploy/olm/storageos/storageos.clusterserviceversion.yaml

# Extract CSV from configmap, update with marketplace changes, removes OLM
# scorecard proxy container and write to the final CSV file.
for target in rhel rhm-1tb rhm-10tb; do
    [ -d deploy/olm/csv-${target} ] || mkdir -p deploy/olm/csv-${target}
    yq r deploy/storageos-operators.configmap.yaml \
        data.clusterServiceVersions | yq r - [0] | \
        yq w -s deploy/olm/${target}-changes.yaml - | \
        yq d - 'spec.install.spec.deployments[0].spec.template.spec.containers[1]' \
        > deploy/olm/csv-${target}/storageos.clusterserviceversion.yaml
done

# Read metadata file configmap and update the CRD files.

# storageoscluster
yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | yq r - [0] | tee \
    deploy/crds/storageos_v1_storageoscluster_crd.yaml \
    deploy/olm/storageos/storageoscluster.crd.yaml > /dev/null
# job
yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | yq r - [1] | tee \
    deploy/crds/storageos_v1_job_crd.yaml \
    deploy/olm/storageos/storageosjob.crd.yaml > /dev/null
# upgrade
yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | yq r - [2] | tee \
    deploy/crds/storageos_v1_storageosupgrade_crd.yaml \
    deploy/olm/storageos/storageosupgrade.crd.yaml > /dev/null
# nfsserver
yq r deploy/storageos-operators.configmap.yaml \
    data.customResourceDefinitions | yq r - [3] | tee \
    deploy/crds/storageos_v1_nfsserver_crd.yaml \
    deploy/olm/storageos/storageosnfsserver.crd.yaml > /dev/null

# Extract package from configmap, update and write to the final file.
yq r deploy/storageos-operators.configmap.yaml \
    data.packages | yq r - [0] | yq w -s deploy/olm/package-changes.yaml - > \
    deploy/olm/storageos/storageos.package.yaml
