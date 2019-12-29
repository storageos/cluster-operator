#!/bin/bash
set -e

# This script uses github hub https://hub.github.com/ to create a pull request.

# Env vars definitions:-
# 
# GITHUB_USER: github username
# GITHUB_EMAIL: github user email
# GITHUB_TOKEN: github user API token with repo access permission only
# SIGN_OFF_NAME: git commit sign-off name
# VERSION: release version
# TARGET_REPO: upstream community operator repo
# COMMUNITY_REPO_PATH: community operator repo path
# COMMUNITY_PKG_PATH: target operator dir path in the community operator repo
# OLM_ROOT: path to the directory containing CRD, CSV and package file.


declare -a metadatafiles=(
  "${OLM_ROOT}/storageoscluster.crd.yaml"
  "${OLM_ROOT}/storageosjob.crd.yaml"
  "${OLM_ROOT}/storageosupgrade.crd.yaml"
  "${OLM_ROOT}/storageosnfsserver.crd.yaml"
  "${OLM_ROOT}/storageos.v${VERSION}.clusterserviceversion.yaml"
)

# Setup netrc.
echo "machine github.com
  login $GITHUB_USER
  password $GITHUB_TOKEN
" > ~/.netrc

# Configure git.
git config --global user.email "$GITHUB_EMAIL"
git config --global user.name "$SIGN_OFF_NAME"

# Clone community hub repo.
git clone $TARGET_REPO $COMMUNITY_REPO_PATH

echo "Creating new release dir in community repo for $VERSION"
mkdir $COMMUNITY_REPO_PATH/$COMMUNITY_PKG_PATH/$VERSION

# Copy OLM manifest files.
for i in "${metadatafiles[@]}"
do
  echo "Copying $i"
  # Copy the metada files to target package in community repo.
  # Example: cp deploy/olm/storageos/storageos.package.yaml /go/src/github.com/operator-framework/community-operators/upstream-community-operators/storageos/
  cp $i $COMMUNITY_REPO_PATH/$COMMUNITY_PKG_PATH/$VERSION
done

# Copy OLM package file.
cp $OLM_ROOT/storageos.package.yaml $COMMUNITY_REPO_PATH/$COMMUNITY_PKG_PATH

# Create branch, commit and create a PR.
MESSAGE="Update StorageOS Operator to version ${VERSION}"
PR_TEMPLATE="### Update to existing Operators

* [x] Is your new CSV pointing to the previous version with the replaces property?
* [x] Is your new CSV referenced in the [appropriate channel](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#bundle-format) defined in the package.yaml ?
* [x] Have you tested an update to your Operator when deployed via OLM?
* [x] Is your submission [signed](https://github.com/operator-framework/community-operators/blob/master/docs/contributing.md#sign-your-work)?"

pushd $COMMUNITY_REPO_PATH
hub remote add fork https://github.com/$GITHUB_USER/community-operators
git checkout -b $VERSION
git add *
git commit -m "$MESSAGE" -s
git push fork $VERSION
# Create PR message by combining commit message and PR template.
PR_MESSAGE="${MESSAGE}

${PR_TEMPLATE}"
hub pull-request -m "$PR_MESSAGE"
