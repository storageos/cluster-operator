#!/usr/bin/env bash

set -Eeuxo pipefail

readonly REPO_ROOT="${REPO_ROOT:-$(git rev-parse --show-toplevel)}"

enable_lio() {
    echo "Enable LIO"
    sudo apt -y update
    sudo apt -y install linux-image-extra-$(uname -r)
    sudo mount --make-shared /sys
    sudo mount --make-shared /
    sudo mount --make-shared /dev
    docker run --name enable_lio --privileged --rm --cap-add=SYS_ADMIN -v /lib/modules:/lib/modules -v /sys:/sys:rshared storageos/init:0.1
    echo
}

run_kind() {
    echo "Download kind binary..."
    # docker run --rm -it -v "$(pwd)":/go/bin golang go get sigs.k8s.io/kind && sudo mv kind /usr/local/bin/
    wget -O kind 'https://docs.google.com/uc?export=download&id=1C_Jrj68Y685N5KcOqDQtfjeAZNW2UvNB' --no-check-certificate && chmod +x kind && sudo mv kind /usr/local/bin/

    echo "Download kubectl..."
    curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/"${K8S_VERSION}"/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
    echo

    echo "Create Kubernetes cluster with kind..."
    # kind create cluster --image=kindest/node:"$K8S_VERSION"
    kind create cluster --image storageos/kind-node:v1.13.2

    echo "Export kubeconfig..."
    # shellcheck disable=SC2155
    export KUBECONFIG="$(kind get kubeconfig-path)"
    cp $(kind get kubeconfig-path) ~/.kube/config
    echo

    echo "Get cluster info..."
    kubectl cluster-info
    echo

    echo "Wait for kubernetes to be ready"
    JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done
    echo

    kubectl get all --all-namespaces
}

run_minikube() {
    echo "Install socat and util-linux"
    sudo apt-get install -y socat util-linux
    echo

    echo "Copy nsenter tool for Ubuntu 14.04 (current travisCI build VM version)"
    # shellcheck disable=SC2046
    sudo docker run --rm -v $(pwd):/target jpetazzo/nsenter
    sudo mv -fv nsenter /usr/local/bin/
    echo

    echo "Run minikube"
    # Download kubectl, which is a requirement for using minikube.
    curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/${K8S_VERSION}/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
    # Download minikube.
    curl -Lo minikube https://github.com/kubernetes/minikube/releases/download/${MINIKUBE_VERSION}/minikube-linux-amd64 && chmod +x minikube && sudo mv minikube /usr/local/bin/
    # TODO: remove the --bootstrapper flag once this issue is solved: https://github.com/kubernetes/minikube/issues/2704
    sudo minikube config set WantReportErrorPrompt false
    sudo -E minikube start --vm-driver=none --cpus 2 --memory 4096 --bootstrapper=localkube --kubernetes-version=${K8S_VERSION} --extra-config=apiserver.Authorization.Mode=RBAC

    echo "Enable add-ons..."
    sudo minikube addons disable kube-dns
    sudo minikube addons enable coredns
    echo

    # Fix the kubectl context, as it's often stale.
    # - minikube update-context
    # Wait for Kubernetes to be up and ready.
    JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done
    echo

    echo "Get cluster info..."
    kubectl cluster-info
    echo

    echo "Create cluster admin..."
    kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
    echo
}

run_openshift() {
    echo "Run openshift"
    # Configure insecure docker registry for openshift
    sudo service docker stop
    sudo sed -i 's/DOCKER_OPTS=\"/DOCKER_OPTS=\"--insecure-registry 172.30.0.0\/16 /' /etc/default/docker
    sudo service docker start

    # Change directory to $HOME to avoid creating openshift cluster files in the project git repo.
    # This is needed to avoid including the cluster files in docker build context.
    cd $HOME
    # Download oc to spin up openshift on local docker instance
    curl -Lo oc.tar.gz https://github.com/openshift/origin/releases/download/v3.11.0/openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit.tar.gz
    # Put oc binary in path
    tar xvzOf oc.tar.gz openshift-origin-client-tools-v3.11.0-0cbc58b-linux-64bit/oc > oc && chmod +x oc && sudo mv oc /usr/local/bin/
    # Start oc cluster
    oc cluster up
    # Become cluster admin
    oc login -u system:admin
    # Install kubectl
    curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.11.3/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
    # Change directory to the project directory.
    cd -
    echo
}

install_operatorsdk() {
    echo "Install operator-sdk"
    make install-operator-sdk
    echo
}

# Prints log for all pods in the specified namespace.
# Args:
#   $1 The namespace
print_pod_details_and_logs() {
    local namespace="${1?Namespace is required}"

    kubectl get pods --show-all --no-headers --namespace "$namespace" | awk '{ print $1 }' | while read -r pod; do
        if [[ -n "$pod" ]]; then
            printf '\n================================================================================\n'
            printf ' Details from pod %s\n' "$pod"
            printf '================================================================================\n'

            printf '\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n'
            printf ' Description of pod %s\n' "$pod"
            printf '~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n'

            kubectl describe pod --namespace "$namespace" "$pod" || true

            printf '\n~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n'
            printf ' End of description for pod %s\n' "$pod"
            printf '~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~\n'

            local init_containers
            init_containers=$(kubectl get pods --show-all --output jsonpath="{.spec.initContainers[*].name}" --namespace "$namespace" "$pod")
            for container in $init_containers; do
                printf -- '\n--------------------------------------------------------------------------------\n'
                printf ' Logs of init container %s in pod %s\n' "$container" "$pod"
                printf -- '--------------------------------------------------------------------------------\n\n'

                kubectl logs --namespace "$namespace" --container "$container" "$pod" || true

                printf -- '\n--------------------------------------------------------------------------------\n'
                printf ' End of logs of init container %s in pod %s\n' "$container" "$pod"
                printf -- '--------------------------------------------------------------------------------\n'
            done

            local containers
            containers=$(kubectl get pods --show-all --output jsonpath="{.spec.containers[*].name}" --namespace "$namespace" "$pod")
            for container in $containers; do
                printf '\n--------------------------------------------------------------------------------\n'
                printf -- ' Logs of container %s in pod %s\n' "$container" "$pod"
                printf -- '--------------------------------------------------------------------------------\n\n'

                kubectl logs --namespace "$namespace" --container "$container" "$pod" || true

                printf -- '\n--------------------------------------------------------------------------------\n'
                printf ' End of logs of container %s in pod %s\n' "$container" "$pod"
                printf -- '--------------------------------------------------------------------------------\n'
            done

            printf '\n================================================================================\n'
            printf ' End of details for pod %s\n' "$pod"
            printf '================================================================================\n\n'
        fi
    done
}

main() {
    enable_lio

    # Install minikube or openshift on the test machine based on the passed args.
    if [ "$1" = "minikube" ]; then
        run_minikube
    elif [ "$1" = "openshift" ]; then
        run_openshift
        # # TODO: Add node label for master node. This is required by the OLM
        # # deployments to work in the next version of OLM 0.9.
        # kubectl label nodes localhost node-role.kubernetes.io/master=

        # Update CR with k8sDistro set to openshift
        yq w -i deploy/storageos-operators.olm.cr.yaml spec.k8sDistro openshift
    elif [ "$1" = "kind" ]; then
        run_kind
    fi

    install_operatorsdk

    echo "Ready for testing"

    NODE_NAME=$(kubectl get nodes --no-headers=true -o=name)

    # Build the operator container image.
    # This would build a container with tag storageos/cluster-operator:test,
    # which is used in the e2e test setup below.
    make image/cluster-operator

    # Move the operator container inside Kind container so that the image is
    # available to the docker in docker environment.
    if [ "$1" = "kind" ]; then
        x=$(docker ps -f name=kind-1-control-plane -q)
        docker save storageos/cluster-operator:test > cluster-operator.tar
        docker cp cluster-operator.tar $x:/cluster-operator.tar
        docker exec $x bash -c "docker load < /cluster-operator.tar"
    fi

    if [ "$2" = "olm" ]; then
        # Lint the OLM CSV bundle.
        make olm-lint

        # Create and lint the bundle for openshift metadata scanner.
        make metadata-bundle-lint

        source ./deploy/olm/olm.sh
        # Not using quick install here because the order in which the resources
        # are created is unreliable and results in flaky test setup. Hard to
        # reproduce it locally. The errors are mostly due to the CRD being
        # used before it's created.
        install_olm

        # Wait for all the OLM resources to be created and ready.
        sleep 10

        install_storageos_operator
        install_storageos
    else
        # Add taint on the node.
        kubectl taint nodes $NODE_NAME key=value:NoSchedule

        # Create a namespace for testing operator.
        # This is needed because the service account created using
        # deploy/service_account.yaml has a static namespace. Creating operator in
        # other namespace will result in permission errors.
        kubectl create ns storageos-operator

        # Run the e2e test in the created namespace.
        #
        # Tags are passed to test local command to run e2e test packages only with
        # specific config.
        # Tag "intree" would run k8s intree plugin test setup.
        # Tag "csi" would run csi test setup.
        # The cluster-operator container image used in the e2e setup is based on the
        # operator container image in the manifest file deploy/operator.yaml. The
        # deploy manifests are combined and applied to deploy the operator before
        # running the e2e tests.
        #
        # NOTE: Append this test command with `|| true` to debug by inspecting the
        # resource details. Also comment `defer ctx.Cleanup()` in the cluster to
        # avoid resouce cleanup.
        operator-sdk test local ./test/e2e --go-test-flags "-v -tags intree" --namespace storageos-operator

        echo "Deleting namespace storageos..."
        kubectl delete ns storageos --ignore-not-found=true

        # TODO: Remove these manual SCC permissions. It works automatically in
        # the OLM tests and in manual test runs, but fails in CI only with
        # no statefulset pods created. Try to reproduce it and find what's
        # causing this issue.
        if [ "$1" = "openshift" ]; then
        # Add storageos service account to Security Context Constraint (SCC).
        # This is openshift specific permission which is required for the operator
        # to work.
        oc adm policy add-scc-to-user privileged system:serviceaccount:storageos:storageos-daemonset-sa
        oc adm policy add-scc-to-user privileged system:serviceaccount:storageos:storageos-statefulset-sa
        fi

        operator-sdk test local ./test/e2e --go-test-flags "-v -tags csi" --namespace storageos-operator

        echo "Deleting namespace storageos..."
        kubectl delete ns storageos --ignore-not-found=true

        # echo "**** Resource details for storageos-operator namespace ****"
        # print_pod_details_and_logs storageos-operator

        # echo "**** Resource details for storageos namespace ****"
        # print_pod_details_and_logs storageos
    fi

    echo "Done Testing!"
}

main "$@"
