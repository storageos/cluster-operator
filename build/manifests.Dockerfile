# Nixery image registry generates images on the fly with the counted dependencies.
FROM nixery.dev/shell/remake/go_1_15/cacert/git as build

ENV PATH=/share/go/bin:$PATH
ENV GOPATH=/share/go
ENV GIT_SSL_CAINFO=/etc/ssl/certs/ca-bundle.crt

WORKDIR /tmp/src

COPY . .

ARG OPERATOR_IMAGE=storageos/cluster-operator:develop
RUN remake go-gen install-manifest

# Create the final image.

FROM nixery.dev/shell

COPY --from=build /tmp/src/storageos-operator.yaml /operator.yaml

ENTRYPOINT cat /operator.yaml