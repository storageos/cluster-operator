OPERATOR_IMAGE ?= storageos/cluster-operator:test
GO_BUILD_CMD = go build -v
GO_ENV = GOOS=linux CGO_ENABLED=0
SDK_VERSION = v0.17.2
MACHINE = $(shell uname -m)
BUILD_IMAGE = golang:1.15.2
BASE_IMAGE = registry.access.redhat.com/ubi8/ubi-minimal
BUILD_DIR = "build"
OPERATOR_SDK = $(BUILD_DIR)/operator-sdk
YQ = $(BUILD_DIR)/yq
YQ3 = $(BUILD_DIR)/yq3
GOLANGCI_LINT = $(BUILD_DIR)/golangci-lint
OUTPUT_DIR = $(BUILD_DIR)/_output

OPERATOR_NAME = storageosoperator
# Directory for all the operator-metadata bundle files.
BUNDLE_DIR = $(PWD)/bundle
# Current bundle version is the next release version.
# Bundle generation requires the current version number of the operator.
CURRENT_BUNDLE_VERSION = "2.4.0"
SERVICE_ACCOUNT_NAME = $(shell $(YQ3) r deploy/operator.yaml spec.template.spec.serviceAccountName)
SERVICE_ACCOUNT_PATH = "spec.install.spec.clusterPermissions[0].serviceAccountName"
CSV_FILE = "bundle/manifests/storageosoperator.clusterserviceversion.yaml"

# Set the new version before running the release target.
NEW_VERSION ?= v2.3.0

# Variables to be used when running the make targets in a container.
CACHE_DIR = $(shell pwd)/.cache
PROJECT = github.com/storageos/cluster-operator
GOARCH ?= amd64
GO_VERSION = 1.15.2

# Since go modules don't allow non-go files to be vendored, the code generator
# scripts needed for updating the generated codes are downloaded in the cache
# dir.
K8S_CODE_GEN_DIR = k8s.io/code-generator
CACHE_K8S_CODE_GEN_DIR = $(CACHE_DIR)/go/src/$(K8S_CODE_GEN_DIR)
K8S_GEN_GROUPS_SCRIPT = $(K8S_CODE_GEN_DIR)/generate-groups.sh
CACHE_K8S_GEN_GROUPS_SCRIPT = $(CACHE_DIR)/go/src/$(K8S_GEN_GROUPS_SCRIPT)
CACHE_K8S_DIR = $(CACHE_DIR)/go/src/k8s.io

# When this file name is modified, the new name must be added in .travis.yml
# file as well for publishing the file at release.
METADATA_FILE = storageos-olm-metadata.zip

LDFLAGS += -X github.com/storageos/cluster-operator/pkg/controller/storageosupgrade.operatorImage=$(OPERATOR_IMAGE)

.DEFAULT_GOAL:=help

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-20s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
	@echo ""
	@echo "To run any of the above in docker, suffix the command with '-docker':"
	@echo ""
	@echo "  make operator-docker"
	@echo ""

##############################
# Development                #
##############################

##@ Development

.PHONY: all operator upgrader go-gen generate metadata-update local-run lint olm-lint metadata-bundle-lint clean

all: lint unittest operator

upgrader:
	@echo "Building upgrader"
	$(GO_ENV) $(GO_BUILD_CMD) -mod=vendor \
		-o $(OUTPUT_DIR)/bin/upgrader \
		./cmd/upgrader

operator: upgrader ## Build operator binaries.
	@echo "Building cluster-operator"
	$(GO_ENV) $(GO_BUILD_CMD) -mod=vendor -ldflags "$(LDFLAGS)" \
		-o $(OUTPUT_DIR)/bin/cluster-operator \
		./cmd/manager

k8s-code-gen:
	echo "checking code-gen in cache"
	@if [ ! -f $(CACHE_K8S_GEN_GROUPS_SCRIPT) ]; then \
		echo "k8s code-gen generate-groups.sh not found, downloading the code-gen repo..." && \
		rm -rf $(CACHE_K8S_CODE_GEN_DIR) && \
		mkdir -p $(CACHE_K8S_DIR) && \
		git clone --depth=1 https://github.com/kubernetes/code-generator $(CACHE_K8S_CODE_GEN_DIR); \
	fi

# Generate APIs, CRD specs and CRD clientset.
go-gen: operator-sdk k8s-code-gen
	# generate k8s requires GOROOT to be set.
	GOROOT=$(GOPATH) GO111MODULE=on $(OPERATOR_SDK) generate k8s
	GO111MODULE=on $(OPERATOR_SDK) generate crds
	# TODO: Install kube-openapi and generate OpenAPI. Operator-sdk no
	# longer provides subcommand to generate OpenAPI. Install
	# k8s.io/kube-openapi
	# Generate OpenAPI.
	# openapi-gen --logtostderr=true \
        #           -i ./pkg/apis/storageos/v1 \
        #           -o "" \
        #           -O zz_generated.openapi \
        #           -p ./pkg/apis/storageos/v1 \
        #           -h $(CACHE_DIR)/go/src/k8s.io/code-generator/hack/boilerplate.go.txt \
        #           -r "-"
	# Generate storageos operator resource client.
	$(CACHE_K8S_GEN_GROUPS_SCRIPT) "deepcopy,client" \
		github.com/storageos/cluster-operator/pkg/client \
		github.com/storageos/cluster-operator/pkg/apis storageos:v1

generate: go-gen-docker ## Run all the code generators.

metadata-update: yq ## Update all the OLM metadata files.
	# Update all the metadata files in-place.
	bash scripts/metadata-checker/update-metadata-files.sh

# Run operator locally, from outside of the k8s cluster.
local-run: upgrader ## Run the opereator locally, out of k8s.
	OPERATOR_NAME=cluster-operator \
		      $(OPERATOR_SDK) run --local
	# OPERATOR_NAME=cluster-operator operator-sdk run --local --go-ldflags "$(LDFLAGS)"

lint: golangci-lint ## Lint the code.
	$(GOLANGCI_LINT) run --timeout 3m

# Lint the OLM metadata bundle.
olm-lint: yq generate ## Lint the OLM related files.
	# Generate metadata files and verify all the metadata files are up-to-date.
	bash scripts/metadata-checker/metadata-diff-checker.sh
	# Verify the OLM metada using operator-courier.
	docker run -it --rm \
		-v $(PWD)/deploy/olm/storageos/:/storageos \
		-v $(PWD)/deploy/olm/csv-rhel/:/rhel \
		python:3 bash -c "pip install operator-courier && operator-courier verify --ui_validate_io /storageos"

# Create a metadata zip file and lint the bundle.
metadata-bundle-lint: metadata-zip ## Generate a metadata-bundle and lint it.
	docker run -it --rm -v $(PWD)/$(OUTPUT_DIR)/:/metadata \
		-w /home/test/ \
		python:3 bash -c "pip install operator-courier && unzip /metadata/$(METADATA_FILE) -d out && operator-courier --verbose verify --ui_validate_io out/"

tidy: ## Prune, add and vendor go dependencies.
	go mod tidy -v
	go mod vendor -v

clean: ## Clean all the generated artifacts.
	rm -rf $(OUTPUT_DIR) storageos-operator.yaml

##############################
# Images                     #
##############################

##@ Images

.PHONY: operator-image dev-image

operator-image: operator-sdk ## Build the operator image for distribution.
	docker build \
		--build-arg BUILD_IMAGE=$(BUILD_IMAGE) \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg OPERATOR_IMAGE=$(OPERATOR_IMAGE) \
		. -f build/Dockerfile -t $(OPERATOR_IMAGE)

dev-image: operator-sdk operator-docker ## Build an image quickly for testing (for development).
	docker build . -f build/dev/Dockerfile -t $(OPERATOR_IMAGE)

##############################
# Third-party tools          #
##############################

##@ Third-party tools

.PHONY: operator-sdk yq yq3 golangci-lint

operator-sdk: ## Download operator-sdk.
	# Download sdk only if it's not available.
	@if [ ! -f $(OPERATOR_SDK) ]; then \
		curl -Lo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/$(SDK_VERSION)/operator-sdk-$(SDK_VERSION)-$(MACHINE)-linux-gnu && \
		chmod +x $(OPERATOR_SDK); \
	fi

yq: ## Install yq.
	@if [ ! -f $(YQ) ]; then \
		curl -Lo $(YQ) https://github.com/mikefarah/yq/releases/download/2.3.0/yq_linux_amd64 && \
		chmod +x $(YQ); \
	fi

yq3: ## Install yq3.
	@if [ ! -f $(YQ3) ]; then \
		curl -Lo $(YQ3) https://github.com/mikefarah/yq/releases/download/3.4.1/yq_linux_amd64 && \
		chmod +x $(YQ3); \
	fi

golangci-lint: ## Install golangci-lint
	@if [ ! -f $(GOLANGCI_LINT) ]; then \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(BUILD_DIR) v1.27.0; \
	fi

##############################
# Tests                      #
##############################

##@ Tests

.PHONY: unittest scorecard-test

unittest: ## Run all the unit tests.
	go test -mod=vendor -v -race `go list -v ./... | grep -v test/e2e | grep -v olm` -count=1

# Runs the operator-sdk scorecard tests. Expects the operator to be installed
# using OLM first.
scorecard-test: ## Run Operator scorecard test.
	bash test/scorecard-test.sh

##############################
# Release                    #
##############################

##@ Release

.PHONY: release metadata-zip install-manifest

# Prepare the repo for a new release.
release: yq ## Prepare for a new release. Pass NEW_VERSION with the next release version number.
	bash scripts/release-helpers/release-gen.sh $(NEW_VERSION)

# Generate metadata bundle for openshift metadata scanner.
metadata-zip: ## Generate OLM metadata-zip bundle (in package manifest format)
	# Remove any existing metadata bundle.
	rm -f $(OUTPUT_DIR)/$(METADATA_FILE)
	# Ensure the target path exists.
	mkdir -p $(OUTPUT_DIR)
	# -j strips the parent directories and adds the files at the root. This is
	# a requirement for the openshift metadata scanner.
	zip -j $(OUTPUT_DIR)/$(METADATA_FILE) \
		deploy/olm/storageos/storageos.package.yaml \
		deploy/olm/storageos/storageoscluster.crd.yaml \
		deploy/olm/storageos/storageosjob.crd.yaml \
		deploy/olm/storageos/storageosupgrade.crd.yaml \
		deploy/olm/storageos/storageosnfsserver.crd.yaml \
		deploy/olm/csv-rhel/storageos.v*.clusterserviceversion.yaml

bundle: operator-sdk yq3 ## Generate operator metadata in bundle format.
	$(OPERATOR_SDK) generate csv \
		--update-crds \
		--csv-version=$(CURRENT_BUNDLE_VERSION) \
		--operator-name=$(OPERATOR_NAME) \
		--output-dir=$(BUNDLE_DIR)
	# Fix the cluster permission service account name in the generated
	# bundle.
	# Refer: https://github.com/operator-framework/operator-sdk/pull/3610
	# The fix isn't available for operator-sdk < v0.19.0.
	$(YQ3) w -i $(CSV_FILE) $(SERVICE_ACCOUNT_PATH) $(SERVICE_ACCOUNT_NAME)


# Generates a single manifest for installing the operator.
install-manifest: yq3 ## Generate operator install manifest file.
	bash scripts/create-manifest.sh $(OPERATOR_IMAGE)



# This target matches any target ending in '-docker' eg. 'unittest-docker'. This
# allows running makefile targets inside a container by appending '-docker' to
# it.
%-docker:
	# k8s code-generator's generate-groups.sh script expects a boilerplate
	# file to exist under the code-generator project in GOPATH. The
	# generate-groups script is vendored. Create an empty boilerplate file
	# at the expected location.
	mkdir -p $(CACHE_DIR)/go $(CACHE_DIR)/cache $(CACHE_DIR)/go/src/k8s.io/code-generator/hack/
	touch $(CACHE_DIR)/go/src/k8s.io/code-generator/hack/boilerplate.go.txt
	# golangci-lint build cache.
	mkdir -p $(CACHE_DIR)/golangci-lint
	# Run the make target in docker.
	docker run -it --rm \
		-v $(CACHE_DIR)/go:/go \
		-v $(CACHE_DIR)/cache:/.cache/go-build \
		-v $(CACHE_DIR)/golangci-lint:/.cache/golangci-lint \
		-v $(shell pwd):/go/src/${PROJECT} \
		-w /go/src/${PROJECT} \
		-u $(shell id -u):$(shell id -g) \
		-e GOARCH=$(GOARCH) \
		--entrypoint "make" \
		golang:$(GO_VERSION) \
		"$(patsubst %-docker,%,$@)"
