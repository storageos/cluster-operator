ARG BUILD_IMAGE=golang:1.11.5
ARG BASE_IMAGE=alpine:3.9
ARG OPERATOR_IMAGE=storageos/cluster-operator:test

FROM ${BUILD_IMAGE} AS build
# OPERATOR_IMAGE needs to be passed to build/cluster-operator for constructing
# the ldflags.
ARG OPERATOR_IMAGE
WORKDIR /go/src/github.com/storageos/cluster-operator/
COPY . /go/src/github.com/storageos/cluster-operator/
RUN cp build/operator-sdk /usr/local/bin/
RUN make generate
RUN make build/cluster-operator OPERATOR_IMAGE=$OPERATOR_IMAGE
RUN make build/upgrader

FROM ${BASE_IMAGE}
RUN apk --no-cache add openssl wget
# Docker is required by the upgrader.
RUN \
    wget --no-check-certificate -q https://download.docker.com/linux/static/stable/x86_64/docker-17.03.0-ce.tgz && \
    tar -xvzf docker-17.03.0-ce.tgz && \
    cp docker/docker /bin/ && \
    rm -rf docker* && \
    chmod +x /bin/docker
COPY --from=build /go/src/github.com/storageos/cluster-operator/build/_output/bin/cluster-operator /usr/local/bin/cluster-operator
COPY --from=build /go/src/github.com/storageos/cluster-operator/build/_output/bin/upgrader /usr/local/bin/upgrader
COPY --from=build /go/src/github.com/storageos/cluster-operator/cmd/image-puller/docker-puller.sh /usr/local/bin/docker-puller.sh
