# This can be used to install a specific version of OLM.
olm_quick_install() {
    echo "Quick Install OLM"

    # Use the release manifest.
    kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.12.0/crds.yaml
    kubectl apply -f https://github.com/operator-framework/operator-lifecycle-manager/releases/download/0.12.0/olm.yaml

    # Create this cluster role binding to grant permissions to the olm web console.
    kubectl create clusterrolebinding cluster-admin-binding --clusterrole=cluster-admin --user=system:serviceaccount:kube-system:default

    echo
}

install_storageos_operator() {
    echo "Install StorageOS Operator via OLM"

    # Install storageos catalog configmap.
    kubectl create -f deploy/storageos-operators.configmap.yaml
    # Install storageos catalog source.
    kubectl create -f deploy/storageos-operators.catalogsource.yaml
    # Install storageos operator by creating a subscription.
    kubectl create -f deploy/storageos-operators.subscription.yaml

    # Wait for storageos operator to be ready.
    until kubectl -n olm get deployment storageos-operator --no-headers -o go-template='{{.status.readyReplicas}}' | grep -q 1; do sleep 3; done

    echo
}

# Install storageos with default CSI helpers(StatefulSet).
install_storageos() {
    echo "Install StorageOS"

    kubectl apply -f deploy/storageos-operators.olm.cr.yaml
    sleep 5

    kubectl -n storageos get all

    echo "Waiting for storageos daemonset to be ready"
    until kubectl -n storageos get daemonset storageos-daemonset --no-headers -o go-template='{{.status.numberReady}}' | grep -q 1; do sleep 5; done
    echo "Daemonset ready!"

    echo "Waiting for storageos statefulset to be ready"
    until kubectl -n storageos get statefulset storageos-statefulset --no-headers -o go-template='{{.status.readyReplicas}}' | grep -q 1; do sleep 5; done
    echo "Statefulset ready!"
}

# Install storageos with CSI helpers as Deployment.
install_storageos_csi_deployment() {
    echo "Install StorageOS with CSI "

    # Create the api secret.
    yq r -d1 deploy/storageos-operators.olm.cr.yaml | kubectl apply -f -

    # Read the cluster manifest, set csi.deploymentStrategy to "deployment" and
    # create.
    yq r -d0 deploy/storageos-operators.olm.cr.yaml | \
        yq w - spec.csi.deploymentStrategy deployment | kubectl apply -f -
    sleep 5

    kubectl -n storageos get all

    echo "Waiting for storageos daemonset to be ready"
    until kubectl -n storageos get daemonset storageos-daemonset --no-headers -o go-template='{{.status.numberReady}}' | grep -q 1; do sleep 5; done
    echo "Daemonset ready!"

    echo "Waiting for storageos CSI helper Deployment to be ready"
    until kubectl -n storageos get deployment storageos-csi-helper --no-headers -o go-template='{{.status.readyReplicas}}' | grep -q 1; do sleep 5; done
    echo "CSI helper Deployment ready!"
}

uninstall_storageos() {
    echo "Uninstalling StorageOS"
    kubectl delete -f deploy/storageos-operators.olm.cr.yaml
    # Allow the resources to be deleted by the operator.
    sleep 10
    # Delete namespace to ensure nothing is left behind.
    kubectl delete ns storageos --ignore-not-found=true
    echo
}

uninstall_storageos_operator() {
    echo "Uninstall StorageOS Operator"

    # Delete the current operator instance.
    kubectl -n olm delete csv storageosoperator.0.0.0

    # Delete all the related resources.
    kubectl delete -f deploy/storageos-operators.subscription.yaml
    kubectl delete -f deploy/storageos-operators.catalogsource.yaml
    kubectl delete -f deploy/storageos-operators.configmap.yaml

    echo
}

uninstall_olm_quick() {
    echo "Uninstalling OLM"
    kubectl delete -f https://raw.githubusercontent.com/operator-framework/operator-lifecycle-manager/master/deploy/upstream/quickstart/olm.yaml
    echo
}
