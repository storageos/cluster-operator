OPERATOR_IMAGE ?= storageos/cluster-operator:test
GO_BUILD_CMD = go build -v
GO_ENV = GOOS=linux CGO_ENABLED=0

LDFLAGS += -X github.com/storageos/cluster-operator/pkg/controller/storageosupgrade.operatorImage=$(OPERATOR_IMAGE)

build/upgrader:
	@echo "Building upgrader"
	GOOS=linux CGO_ENABLED=0 $(GO_BUILD_CMD) \
		-o ./build/_output/bin/upgrader \
		./cmd/upgrader

image/cluster-operator: build/upgrader
	operator-sdk generate k8s
	$(GO_ENV) $(GO_BUILD_CMD) -ldflags "$(LDFLAGS)" \
		-o ./build/_output/bin/cluster-operator \
		./cmd/manager
	# Build the final container
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
