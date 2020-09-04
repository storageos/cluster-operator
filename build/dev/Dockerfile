FROM storageos/base-image:0.2.2
WORKDIR /home/operator
RUN chmod -R g+rwX /home/operator
COPY LICENSE /licenses/
COPY build/_output/bin/cluster-operator /usr/local/bin/cluster-operator
COPY build/_output/bin/upgrader /usr/local/bin/upgrader
USER 1001
