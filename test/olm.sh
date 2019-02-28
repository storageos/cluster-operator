install_olm() {
    echo "Install OLM"

    git clone --depth 1 https://github.com/operator-framework/operator-lifecycle-manager ./test/olm

    # Create the following one at a time to avoid resource not found error.
    kubectl create -f ./test/olm/deploy/upstream/manifests/0.8.1/0000_50_olm_00-namespace.yaml
    kubectl create -f ./test/olm/deploy/upstream/manifests/0.8.1/0000_50_olm_01-olm-operator.serviceaccount.yaml
    for num in {02..05}; do kubectl create -f ./test/olm/deploy/upstream/manifests/0.8.1/0000_50_olm_$num*; done

    # TODO: Remove this once the default olm operator container is fixed, and
    # use the default manifest.
    # Error: flag provided but not defined: -writeStatusName
    kubectl create -f ./deploy/0000_50_olm_06-olm-operator.deployment.yaml
    for num in {07..09}; do kubectl create -f ./test/olm/deploy/upstream/manifests/0.8.1/0000_50_olm_$num*; done

    kubectl create -f deploy/olm-operators.configmap.yaml
    sleep 1
    for num in {11,12}; do kubectl create -f ./test/olm/deploy/upstream/manifests/0.8.1/0000_50_olm_$num*; done
    sleep 5
    kubectl create -f ./test/olm/deploy/upstream/manifests/0.8.1/0000_50_olm_13-packageserver.subscription.yaml

    # Install storageos operator by creating a subscription.
    kubectl create -f deploy/stos.subscription.yaml

    # Wait for storageos operator to be ready.
    until kubectl -n olm get deployment storageos-operator --no-headers -o go-template='{{.status.readyReplicas}}' | grep -q 1; do sleep 1; done

    # Create this cluster role binding to grant permissions to the olm web console.
    kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=system:serviceaccount:kube-system:default

    echo
}

install_olm_storageos() {
    echo "Install StorageOS via OLM"

    kubectl apply -f deploy/stos.olm.cr.yaml
    sleep 5

    kubectl -n storageos get all

    echo "Waiting for storageos daemonset to be ready"
    until kubectl -n storageos get daemonset storageos-daemonset --no-headers -o go-template='{{.status.numberReady}}' | grep -q 1; do sleep 5; done
    echo "Daemonset ready!"

    echo "Waiting for storageos statefulset to be ready"
    until kubectl -n storageos get statefulset storageos-statefulset --no-headers -o go-template='{{.status.readyReplicas}}' | grep -q 1; do sleep 5; done
    echo "Statefulset ready!"
}
