ARG BUILD_IMAGE=golang:1.12.5
ARG BASE_IMAGE=storageos/base-image:0.1.0
ARG OPERATOR_IMAGE=storageos/cluster-operator:test

FROM ${BUILD_IMAGE} AS build
# OPERATOR_IMAGE needs to be passed to build/cluster-operator for constructing
# the ldflags.
ARG OPERATOR_IMAGE
WORKDIR /go/src/github.com/storageos/cluster-operator/
COPY . /go/src/github.com/storageos/cluster-operator/
RUN make operator-sdk
RUN make generate
RUN make build/cluster-operator OPERATOR_IMAGE=$OPERATOR_IMAGE
RUN make build/upgrader

FROM ${BASE_IMAGE}
LABEL name="StorageOS Cluster Operator" \
      maintainer="support@storageos.com" \
      vendor="StorageOS" \
      version="1.2.0" \
      release="1" \
      distribution-scope="public" \
      architecture="x86_64" \
      url="https://docs.storageos.com" \
      io.k8s.description="The StorageOS Cluster Operator installs and manages StorageOS within a cluster." \
      io.k8s.display-name="StorageOS Cluster Operator" \
      io.openshift.tags="storageos,storage,operator,pv,pvc,storageclass,persistent,csi" \
      summary="Highly-available persistent block storage for containerized applications." \
      description="StorageOS transforms commodity server or cloud based disk capacity into enterprise-class storage to run persistent workloads such as databases in containers. Provides high availability, low latency persistent block storage. No other hardware or software is required."

# Docker is required by the upgrader to pre-load images.  Only `docker pull` is
# used.  `podman` would be preferred but it's not available in the package repo,
# and there isn't a binary release that we can easily download into the image.
RUN \
    wget --no-check-certificate -q https://download.docker.com/linux/static/stable/x86_64/docker-17.03.0-ce.tgz && \
    tar -xvzf docker-17.03.0-ce.tgz && \
    cp docker/docker /bin/ && \
    rm -rf docker* && \
    chmod +x /bin/docker
RUN mkdir -p /licenses
COPY --from=build /go/src/github.com/storageos/cluster-operator/LICENSE /licenses/
COPY --from=build /go/src/github.com/storageos/cluster-operator/build/_output/bin/cluster-operator /usr/local/bin/cluster-operator
COPY --from=build /go/src/github.com/storageos/cluster-operator/build/_output/bin/upgrader /usr/local/bin/upgrader
COPY --from=build /go/src/github.com/storageos/cluster-operator/cmd/image-puller/docker-puller.sh /usr/local/bin/docker-puller.sh
