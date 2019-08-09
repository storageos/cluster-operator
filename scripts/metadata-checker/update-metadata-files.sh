#!/bin/bash
set -e

# Extract CSV from configmap, update with community operator changes and write
# to the final CSV file.
yq r deploy/storageos-operators.configmap.yaml \
    data.clusterServiceVersions | yq r - [0] | \
    yq w -s deploy/olm/community-changes.yaml - > \
    deploy/olm/storageos/storageos.clusterserviceversion.yaml

# Extract CSV from configmap, update with rhel operator changes and write to
# the final CSV file.
yq r deploy/storageos-operators.configmap.yaml \
    data.clusterServiceVersions | yq r - [0] | \
    yq w -s deploy/olm/rhel-changes.yaml - > \
    deploy/olm/csv-rhel/storageos.clusterserviceversion.yaml


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
