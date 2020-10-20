VERSION     ?= 2.2.0
GIT_USER_ID ?= storageos
GIT_REPO_ID ?= go-api

.DEFAULT_GOAL:=help

.PHONY: help customise

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n\nTargets:\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-13s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

build: generate customise ## Re-generates the API from openapi.yaml.

generate: ## Runs the openapi-generator on openapi.yaml.
	docker run --rm -v `pwd`:/local -u `id -u ${USER}`:`id -g ${USER}` \
		openapitools/openapi-generator-cli:latest generate \
		--strict-spec=true \
		--additional-properties enumClassPrefix=true \
		--additional-properties packageName=api \
		--additional-properties packageVersion=$(VERSION) \
		-i /local/openapi.yaml -g go -o /local

customise:  ## Applies customisations to the generated API.
	sed -i 's/\[]map/map/ig' *.go
	sed -i "s#module github.com/GIT_USER_ID/GIT_REPO_ID#module github.com/${GIT_USER_ID}/${GIT_REPO_ID}/v2#g" go.mod

clean: ## Removes generated files.
	rm -f .openapi-generator-ignore .gitignore git_push.sh .travis.yml *.go go.mod go.sum
	rm -rf .openapi-generator/ api/ docs/


