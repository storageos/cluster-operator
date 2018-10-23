FROM alpine:3.6

RUN adduser -D cluster-operator
USER cluster-operator

ADD tmp/_output/bin/cluster-operator /usr/local/bin/cluster-operator
