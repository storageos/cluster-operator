#!/bin/bash
set -e

# This script is used for publishing latest(release) and develop(canary)
# container images of the operator. Script argument "tagged" must be passed to
# publish a new release and "develop" must be passed to publish a canary image.

if [ "$1" = "tagged" ]; then
    docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:latest"
    docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:${TRAVIS_TAG}"
    docker push "${IMAGE_NAME}:latest" && docker push "${IMAGE_NAME}:${TRAVIS_TAG}"
    # Trigger a rhel container service build.
    curl -X POST -k -H 'Content-Type: application/json' -i https://connect.redhat.com/api/v2/projects/$RH_PID/build --data '{"tag": "'"${IMAGE_TAG}"'"}'
elif [ "$1" = "develop" ]; then
    docker tag "${IMAGE_NAME}:${IMAGE_TAG}" "${IMAGE_NAME}:develop"
    docker push "${IMAGE_NAME}:develop"
fi
