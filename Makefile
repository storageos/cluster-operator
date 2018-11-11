OPERATOR_IMAGE ?= storageos/cluster-operator:test

build/upgrader:
	@echo "Building upgrader"
	GOOS=linux CGO_ENABLED=0 go build -v -o ./build/_output/bin/upgrader ./cmd/upgrader 

image/cluster-operator: build/upgrader
	operator-sdk generate k8s
	operator-sdk build $(OPERATOR_IMAGE)
	# operator-sdk build $(OPERATOR_IMAGE) --ldflags "-X github.com/storageos/cluster-operator/pkg/controller/storageosupgrade.operatorImage=$(OPERATOR_IMAGE)"

local-run: build/upgrader
	OPERATOR_NAME=cluster-operator operator-sdk up local
	# OPERATOR_NAME=cluster-operator operator-sdk up local --ldflags "-X github.com/storageos/cluster-operator/pkg/controller/storageosupgrade.operatorImage=$(OPERATOR_IMAGE)"

clean:
	rm -rf build/_output

lint:
	golint -set_exit_status $(go list ./...)
	go vet ./...

unittest:
	go test -v `go list -v ./... | grep -v test/e2e`
