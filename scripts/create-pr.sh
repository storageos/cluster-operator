#!/bin/bash
set -e

# This script uses github hub https://hub.github.com/ to create a pull request.

# Env vars definitions:-
# 
# GITHUB_USER: github username
# GITHUB_EMAIL: github user email
# GITHUB_TOKEN: github user API token with repo access permission only
# VERSION: release version
# TARGET_REPO: upstream community operator repo
# COMMUNITY_REPO_PATH: community operator repo path
# COMMUNITY_PKG_PATH: target operator dir path in the community operator repo
# OLM_ROOT: path to the directory containing CRD, CSV and package file.


declare -a metadatafiles=(
  "${OLM_ROOT}/storageos.package.yaml" 
  "${OLM_ROOT}/storageoscluster.crd.yaml"
  "${OLM_ROOT}/storageosjob.crd.yaml"
  "${OLM_ROOT}/storageosupgrade.crd.yaml"
  "${OLM_ROOT}/storageos.v${VERSION}.clusterserviceversion.yaml"
)

# Setup netrc.
echo "machine github.com
  login $GITHUB_USER
  password $GITHUB_TOKEN
" > ~/.netrc

# Configure git.
git config --global user.email "$GITHUB_EMAIL"
git config --global user.name "$GITHUB_USER"

# Clone community hub repo.
git clone $TARGET_REPO $COMMUNITY_REPO_PATH

# Copy OLM package changes.
for i in "${metadatafiles[@]}"
do
  echo "Copying $i"
  # Copy the metada files to target package in community repo.
  # Example: cp deploy/olm/storageos/storageos.package.yaml /go/src/github.com/operator-framework/community-operators/upstream-community-operators/storageos/
  cp $i $COMMUNITY_REPO_PATH/$COMMUNITY_PKG_PATH
done

# Create branch, commit and create a PR.
MESSAGE="Update StorageOS Operator to version ${VERSION}"
pushd $COMMUNITY_REPO_PATH
hub remote add fork https://github.com/$GITHUB_USER/community-operators
git checkout -b $VERSION
git add *
git commit -m "$MESSAGE"
git push fork $VERSION
hub pull-request -m "$MESSAGE"
