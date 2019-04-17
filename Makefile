OPERATOR_IMAGE ?= storageos/cluster-operator:test
GO_BUILD_CMD = go build -v
GO_ENV = GOOS=linux CGO_ENABLED=0
SDK_VERSION = v0.3.0
MACHINE = $(shell uname -m)
BUILD_IMAGE = golang:1.11.5
BASE_IMAGE = storageos/base-image:0.1.0

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
