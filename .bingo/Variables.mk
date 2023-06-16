# Auto generated binary variables helper managed by https://github.com/bwplotka/bingo v0.8. DO NOT EDIT.
# All tools are designed to be build inside $GOBIN.
BINGO_DIR := $(dir $(lastword $(MAKEFILE_LIST)))
GOPATH ?= $(shell go env GOPATH)
GOBIN  ?= $(firstword $(subst :, ,${GOPATH}))/bin
GO     ?= $(shell which go)

# Below generated variables ensure that every time a tool under each variable is invoked, the correct version
# will be used; reinstalling only if needed.
# For example for ginkgo variable:
#
# In your main Makefile (for non array binaries):
#
#include .bingo/Variables.mk # Assuming -dir was set to .bingo .
#
#command: $(GINKGO)
#	@echo "Running ginkgo"
#	@$(GINKGO) <flags/args..>
#
GINKGO := $(GOBIN)/ginkgo-v2.1.4
$(GINKGO): $(BINGO_DIR)/ginkgo.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/ginkgo-v2.1.4"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=ginkgo.mod -o=$(GOBIN)/ginkgo-v2.1.4 "github.com/onsi/ginkgo/v2/ginkgo"

GOLANGCI_LINT := $(GOBIN)/golangci-lint-v1.49.0
$(GOLANGCI_LINT): $(BINGO_DIR)/golangci-lint.mod
	@# Install binary/ries using Go 1.14+ build command. This is using bwplotka/bingo-controlled, separate go module with pinned dependencies.
	@echo "(re)installing $(GOBIN)/golangci-lint-v1.49.0"
	@cd $(BINGO_DIR) && GOWORK=off $(GO) build -mod=mod -modfile=golangci-lint.mod -o=$(GOBIN)/golangci-lint-v1.49.0 "github.com/golangci/golangci-lint/cmd/golangci-lint"

