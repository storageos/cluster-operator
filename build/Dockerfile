FROM alpine:3.6

RUN apk --no-cache add openssl wget
RUN \
    wget --no-check-certificate -q https://download.docker.com/linux/static/stable/x86_64/docker-17.03.0-ce.tgz && \
    tar -xvzf docker-17.03.0-ce.tgz && \
    cp docker/docker /bin/ && \
    rm -rf docker* && \
    chmod +x /bin/docker

ADD build/_output/bin/cluster-operator /usr/local/bin/cluster-operator
ADD build/_output/bin/upgrader /usr/local/bin/upgrader
ADD cmd/image-puller/docker-puller.sh /usr/local/bin/docker-puller.sh
