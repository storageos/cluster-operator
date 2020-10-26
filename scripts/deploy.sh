#!/bin/bash
set -e

# This script is used for publishing latest(release) and develop(canary)
# container images of the operator, triggering rhel container build service and
# create a PR to update community operator version. Script argument "tagged"
# must be passed to publish a new release and "develop" must be passed to
# publish a canary image.

if [ "$1" = "tagged" ]; then
    docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:latest"
    docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:${TRAVIS_TAG}"
    docker push "${IMAGE_NAME}:latest" && docker push "${IMAGE_NAME}:${TRAVIS_TAG}"

    # Trigger a rhel container service build.
    # curl -X POST -k -H 'Content-Type: application/json' -i https://connect.redhat.com/api/v2/projects/$RH_PID/build --data '{"tag": "'"${TRAVIS_TAG}"'"}'

    # Create a PR to community-operator repo.
    docker run --rm -ti \
       -v $PWD:/go/src/github.com/storageos/cluster-operator \
       -e GITHUB_USER=$GH_USER \
       -e GITHUB_EMAIL=$GH_EMAIL \
       -e GITHUB_TOKEN=$API_TOKEN \
       -e SIGN_OFF_NAME=$SIGN_OFF_NAME \
       -e VERSION=$TRAVIS_TAG \
       -e TARGET_REPO="https://github.com/operator-framework/community-operators/" \
       -e COMMUNITY_REPO_PATH="/go/src/github.com/operator-framework/community-operators/" \
       -e COMMUNITY_PKG_PATH="upstream-community-operators/storageos/" \
       -e OLM_ROOT=deploy/olm/storageos \
       -w /go/src/github.com/storageos/cluster-operator \
       tianon/github-hub:2 bash -c "./scripts/create-pr.sh"

elif [ "$1" = "develop" ]; then
    docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:develop"
    docker push "${IMAGE_NAME}:develop"
fi
