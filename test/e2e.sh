#!/usr/bin/env bash

set -Eeuxo pipefail

readonly REPO_ROOT="${REPO_ROOT:-$(git rev-parse --show-toplevel)}"
readonly K8S_1_14="v1.14.2"
readonly K8S_1_13="v1.13.2"
# Two different versions of KinD due to a breaking change between the versions.
readonly KIND_1_14_LINK="https://docs.google.com/uc?export=download&id=1-oy-ui0ZE_T3Fglz1c8ZgnW8U-A4yS8u"
readonly KIND_1_13_LINK="https://docs.google.com/uc?export=download&id=1C_Jrj68Y685N5KcOqDQtfjeAZNW2UvNB"

enable_lio() {
    echo "Enable LIO"
    sudo apt -y update
    sudo apt -y install linux-modules-extra-$(uname -r)
    sudo mount --make-shared /sys
    sudo mount --make-shared /
    sudo mount --make-shared /dev
    docker run --name enable_lio --privileged --rm --cap-add=SYS_ADMIN -v /lib/modules:/lib/modules -v /sys:/sys:rshared storageos/init:0.1
    echo
}

run_kind() {
    echo "Download kind binary..."

    if [ "$1" == "$K8S_1_13" ]; then
        KIND_LINK=$KIND_1_13_LINK
    else
        KIND_LINK=$KIND_1_14_LINK
    fi

    # docker run --rm -it -v "$(pwd)":/go/bin golang go get sigs.k8s.io/kind && sudo mv kind /usr/local/bin/
    wget -O kind $KIND_LINK --no-check-certificate && chmod +x kind && sudo mv kind /usr/local/bin/
    echo "Download kubectl..."
    curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/$1/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
    echo

    echo "Create Kubernetes cluster with kind..."
    # kind create cluster --image=kindest/node:"$K8S_VERSION"
    kind create cluster --image storageos/kind-node:$1 --name kind-1

    echo "Export kubeconfig..."
    # shellcheck disable=SC2155
    export KUBECONFIG="$(kind get kubeconfig-path --name="kind-1")"
    cp $(kind get kubeconfig-path --name="kind-1") ~/.kube/config
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

    # For ubuntu trusty:
    # sudo sed -i 's/DOCKER_OPTS=\"/DOCKER_OPTS=\"--insecure-registry 172.30.0.0\/16 /' /etc/default/docker

    # For ubuntu xenial:
    # Writing directly to /etc/docker/daemon.json seems to leave the file empty.
    # Using a temporary file and move the file after writing to it.
    sudo cat /etc/docker/daemon.json | jq 'setpath(["insecure-registries"]; ["172.30.0.0/16"])' > /tmp/daemon.json && sudo mv /tmp/daemon.json /etc/docker/

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

# Cleanup the resources created by the operator-sdk e2e test.
operator-sdk-e2e-cleanup() {
    echo "Deleting namespace storageos..."
    kubectl delete ns storageos --ignore-not-found=true

    # Delete all the cluster roles.
    kubectl delete clusterrole storageos:csi-attacher \
        storageos:csi-provisioner storageos:driver-registrar \
        storageos:openshift-scc storageos:pod-fencer \
        storageos:scheduler-extender storageos:init \
        --ignore-not-found=true

    # Delete all the cluster role bindings.
    kubectl delete clusterrolebinding storageos:csi-attacher \
        storageos:csi-provisioner storageos:driver-registrar \
        storageos:k8s-driver-registrar storageos:openshift-scc \
        storageos:pod-fencer storageos:scheduler-extender \
        storageos:init --ignore-not-found=true
}

main() {
    enable_lio

    # Install minikube or openshift on the test machine based on the passed args.
    if [ "$1" = "minikube" ]; then
        run_minikube
    elif [ "$1" = "openshift" ]; then
        run_openshift

        # Update CR with k8sDistro set to openshift
        yq w -i deploy/storageos-operators.olm.cr.yaml spec.k8sDistro openshift
    elif [ "$1" = "kind" ]; then
        # OLM installation fails on k8s 1.14 with error "failed to connect
        # service" in CI only. Works fine locally.
        # Refer: https://github.com/operator-framework/operator-lifecycle-manager/issues/740
        # Using k8s 1.13 with old KinD for OLM until that's fixed.
        # New KinD with k8s 1.14 uses containerd and has an incompatible node
        # image.
        if [ "$2" = "olm" ]; then
            run_kind $K8S_1_13
        else
            run_kind $K8S_1_14
        fi
    fi

    install_operatorsdk

    echo "Ready for testing"

    NODE_NAME=$(kubectl get nodes --no-headers=true -o=name)

    # Move the operator container inside Kind container so that the image is
    # available to the docker in docker environment.
    if [ "$1" = "kind" ]; then
        x=$(docker ps -f name=kind-1-control-plane -q)
        # This container image is built separately before running the e2e tests.
        docker save storageos/cluster-operator:test > cluster-operator.tar
        docker cp cluster-operator.tar $x:/cluster-operator.tar

        if [ "$2" = "olm" ]; then
            # kind-olm runs on old KinD with docker.
            # Docker load image from tar archive (KinD with docker).
            docker exec $x bash -c "docker load < /cluster-operator.tar"
        else
            # containerd load image from tar archive (KinD with containerd).
            docker exec $x bash -c "ctr -n k8s.io images import --base-name docker.io/storageos/cluster-operator:test /cluster-operator.tar"
        fi
    fi

    if [ "$2" = "olm" ]; then
        # Lint the OLM CSV bundle.
        make olm-lint

        # Create and lint the bundle for openshift metadata scanner.
        make metadata-bundle-lint

        source ./deploy/olm/olm.sh
        olm_quick_install

        # Wait for all the OLM resources to be created and ready.
        sleep 20

        install_storageos_operator

        # Install storageos with default configurations.
        install_storageos
        uninstall_storageos

        # Install storageos with CSI helpers as Deployment.
        install_storageos_csi_deployment
        uninstall_storageos
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
        # Tag "intree" will run k8s intree plugin test setup.
        # Tag "csi" will run csi test setup with csi helpers as statefulset.
        # Tag "csi-deployment" will run csi test setup with csi helpers as deployment.
        # The cluster-operator container image used in the e2e setup is based on the
        # operator container image in the manifest file deploy/operator.yaml. The
        # deploy manifests are combined and applied to deploy the operator before
        # running the e2e tests.
        #
        # NOTE: Append this test command with `|| true` to debug by inspecting the
        # resource details. Also comment `defer ctx.Cleanup()` in the cluster to
        # avoid resouce cleanup.
        operator-sdk test local ./test/e2e --go-test-flags "-v -tags csi" --namespace storageos-operator
        operator-sdk-e2e-cleanup

        operator-sdk test local ./test/e2e --go-test-flags "-v -tags csideployment" --namespace storageos-operator
        operator-sdk-e2e-cleanup

        operator-sdk test local ./test/e2e --go-test-flags "-v -tags intree" --namespace storageos-operator
        operator-sdk-e2e-cleanup

        # echo "**** Resource details for storageos-operator namespace ****"
        # print_pod_details_and_logs storageos-operator

        # echo "**** Resource details for storageos namespace ****"
        # print_pod_details_and_logs storageos
    fi

    echo "Done Testing!"
}

main "$@"
