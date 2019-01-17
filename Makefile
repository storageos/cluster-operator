OPERATOR_IMAGE ?= storageos/cluster-operator:test
GO_BUILD_CMD = go build -v
GO_ENV = GOOS=linux CGO_ENABLED=0
SDK_VERSION = v0.3.0
MACHINE = $(shell uname -m)

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
	operator-sdk generate k8s

image/cluster-operator: generate build/upgrader build/cluster-operator
	docker build . -f build/Dockerfile -t $(OPERATOR_IMAGE)

local-run: build/upgrader
	OPERATOR_NAME=cluster-operator operator-sdk up local
	# OPERATOR_NAME=cluster-operator operator-sdk up local --go-ldflags "$(LDFLAGS)"

clean:
	rm -rf build/_output

lint:
	golint -set_exit_status $(go list ./...)
	go vet ./...

unittest:
	go test -v `go list -v ./... | grep -v test/e2e`

# Install operator-sdk. To be used in e2e test and deployment stage.
operator-sdk:
	curl -Lo operator-sdk https://github.com/operator-framework/operator-sdk/releases/download/$(SDK_VERSION)/operator-sdk-$(SDK_VERSION)-$(MACHINE)-linux-gnu && chmod +x operator-sdk && sudo mv operator-sdk /usr/local/bin/
