OPERATOR_IMAGE ?= storageos/cluster-operator:test
GO_BUILD_CMD = go build -v
GO_ENV = GOOS=linux CGO_ENABLED=0
SDK_VERSION = v0.8.0
MACHINE = $(shell uname -m)
BUILD_IMAGE = golang:1.12.5
BASE_IMAGE = storageos/base-image:0.1.0

# OLM release versions(StorageOS operator) are required to creating redhat
# operator metadata zip. The zip must have the previous version and the new
# version CSVs. For operator to upgrade, CSV of the previous release, that's
# referred in the replaces field of CSV, must also be included in the zip.
OLM_PREVIOUS_VERSION = 1.1.0
OLM_CURRENT_VERSION = 1.2.0

# When this file name is modified, the new name must be added in .travis.yml
# file as well for publishing the file at release.
METADATA_FILE = storageos-olm-metadata.zip

LDFLAGS += -X github.com/storageos/cluster-operator/pkg/controller/storageosupgrade.operatorImage=$(OPERATOR_IMAGE)

all: lint unittest build/upgrader build/cluster-operator

build/upgrader:
	@echo "Building upgrader"
	$(GO_ENV) $(GO_BUILD_CMD) \
		-o ./build/_output/bin/upgrader \
		./cmd/upgrader

build/cluster-operator:
	@echo "Building cluster-operator"
	$(GO_ENV) $(GO_BUILD_CMD) -ldflags "$(LDFLAGS)" \
		-o ./build/_output/bin/cluster-operator \
		./cmd/manager

generate:
	./build/operator-sdk generate k8s

image/cluster-operator: operator-sdk
	docker build \
		--build-arg BUILD_IMAGE=$(BUILD_IMAGE) \
		--build-arg BASE_IMAGE=$(BASE_IMAGE) \
		--build-arg OPERATOR_IMAGE=$(OPERATOR_IMAGE) \
		. -f build/Dockerfile -t $(OPERATOR_IMAGE)

local-run: build/upgrader
	OPERATOR_NAME=cluster-operator ./build/operator-sdk up local
	# OPERATOR_NAME=cluster-operator operator-sdk up local --go-ldflags "$(LDFLAGS)"

clean:
	rm -rf build/_output

lint:
	golint -set_exit_status $(go list ./...)
	go vet ./...

unittest:
	go test -v -race `go list -v ./... | grep -v test/e2e | grep -v test/olm`

operator-sdk:
	# Download sdk only if it's not available.
	@if [ ! -f build/operator-sdk ]; then \
		curl -Lo build/operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/$(SDK_VERSION)/operator-sdk-$(SDK_VERSION)-$(MACHINE)-linux-gnu && \
		chmod +x build/operator-sdk; \
	fi

# Install operator on a host. Might fail on containers that don't have sudo.
install-operator-sdk: operator-sdk
	sudo cp build/operator-sdk /usr/local/bin/

install-yq:
	@if [ ! -f build/yq ]; then \
		curl -Lo build/yq https://github.com/mikefarah/yq/releases/download/2.3.0/yq_linux_amd64 && \
		chmod +x build/yq; \
	fi

# Generate metadata bundle for openshift metadata scanner.
metadata-zip:
	# Remove any existing metadata bundle.
	rm -f build/_output/$(METADATA_FILE)
	# Ensure the target path exists.
	mkdir -p build/_output/
	# -j strips the parent directories and adds the files at the root. This is
	# a requirement for the openshift metadata scanner.
	zip -j build/_output/$(METADATA_FILE) \
		deploy/olm/storageos/storageos.package.yaml \
		deploy/olm/storageos/storageoscluster.crd.yaml \
		deploy/olm/storageos/storageosjob.crd.yaml \
		deploy/olm/storageos/storageosupgrade.crd.yaml \
		deploy/olm/csv-rhel/storageos.v$(OLM_PREVIOUS_VERSION).clusterserviceversion.yaml \
		deploy/olm/csv-rhel/storageos.v$(OLM_CURRENT_VERSION).clusterserviceversion.yaml

metadata-update:
	# Update all the metadata files in-place.
	bash scripts/metadata-checker/update-metadata-files.sh

# Lint the OLM metadata bundle.
olm-lint:
	# Generate metadata files and verify all the metadata files are up-to-date.
	bash scripts/metadata-checker/metadata-diff-checker.sh
	# Verify the OLM metada using operator-courier.
	docker run -it --rm -v $(PWD)/deploy/olm/storageos/:/storageos \
		python:3 bash -c "pip install operator-courier && operator-courier verify --ui_validate_io /storageos"

# Create a metadata zip file and lint the bundle.
metadata-bundle-lint: metadata-zip
	docker run -it --rm -v $(PWD)/build/_output/:/metadata \
		-w /home/test/ \
		python:3 bash -c "pip install operator-courier && unzip /metadata/$(METADATA_FILE) -d out && operator-courier --verbose verify --ui_validate_io out/"

# Prepare the repo for a new release.
release:
	bash scripts/release-helpers/release-gen.sh $(NEW_VERSION)

# Create a single manifest for installing the operator.
generate-install-manifest: install-yq
	bash scripts/create-manifest.sh $(OPERATOR_IMAGE)
