IMG ?= adapter:latest
CONTAINER_RUNTIME ?= docker
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
KUSTOMIZE ?= $(LOCALBIN)/kustomize
KUSTOMIZE_VERSION ?= v3.8.7
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
GO = GOBIN=$(LOCALBIN) CGO_ENABLED=0 go

KIND ?= $(LOCALBIN)/kind
KUSTOMIZE ?= $(LOCALBIN)/kustomize
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

SRC = internal/entitysource/adapter
CONFIG = $(SRC)/manifests/
API = $(SRC)/api

.PHONY:	all kind-load kind-create kind kustomize api build build-container manifests help run deploy undeploy install uninstall protoc

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

all: help

kind-load: build-container kind
	$(KIND) load docker-image $(IMG)

kind-create: kind
	$(KIND) create cluster
	$(KIND) export kubeconfig

kind: $(KIND)
$(KIND): $(LOCALBIN)
	$(GO) install sigs.k8s.io/kind@latest

kustomize: $(KUSTOMIZE)

$(KUSTOMIZE): $(LOCALBIN)
	rm -fv $(LOCALBIN)/$(KUSTOMIZE)
	curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN)

protoc: $(LOCALBIN)
	$(GO) install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	$(GO) install google.golang.org/protobuf/cmd/protoc-gen-go@latest

codegen: protoc ## generate grpc apis 
	protoc -I $(API) --go-grpc_out=$(API) $(API)/*.proto       
	protoc -I $(API) --go_out=$(API) $(API)/*.proto

build: codegen ## generate apis and build the catalogsource adapter binary
	$(GO) build -o bin/catalogsource_adapter $(SRC)/catalogsource/cmd/cmd.go

build-container: codegen ## build from the dockerfile
	$(CONTAINER_RUNTIME) build -f deppysrc.Dockerfile -t $(IMG) .

run: build ## Build and run the deppysource adapter binary locally
	./bin/catalogsource_adapter

deploy: ## Deploy the adapter on a cluster
	cd $(SRC)/manifests && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build $(SRC)/manifests | kubectl apply -f -

undeploy: ## Delete the adapter from a cluster
	$(KUSTOMIZE) build $(SRC)/manifests | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

install: build-container kind-load deploy ## Build and deploy the adapter on a cluster

uninstall: undeploy ## Uninstall the adapter from a cluster