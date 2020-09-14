# Copyright 2019 The pf9ctl authors.
#
# Usage:
# make                 # builds the artifact
# make ensure          # runs dep ensure which downloads the dependencies
# make clean           # removes the artifact and the vendored packages
# make clean-all       # same as make clean + removes the bin dir which houses dep
# make container-build # build artifact on a Linux based container using golang 1.14.1

SHELL := /usr/bin/env bash
BUILD_NUMBER ?= 10
GITHASH := $(shell git rev-parse --short HEAD)
CWD := $(shell pwd)
PF9_VERSION ?= 1.0.0
VERSION := $(PF9_VERSION)-$(BUILD_NUMBER)
DETECTED_OS := $(shell uname -s)
BIN := pf9ctl
REPO := pf9ctl
PACKAGE_GOPATH := /go/src/github.com/platform9/$(REPO)
LDFLAGS := $(shell source ./version.sh ; KUBE_ROOT=. ; KUBE_GIT_VERSION=${VERSION_OVERRIDE} ; kube::version::ldflags)
GIT_STORAGE_MOUNT := $(shell source ./git_utils.sh; container_git_storage_mount)

.PHONY: clean clean-all container-build default format test

default: $(BIN)

container-build:
	docker run --rm -e VERSION_OVERRIDE=${VERSION_OVERRIDE} -v $(PWD):$(PACKAGE_GOPATH) $(GIT_STORAGE_MOUNT) -w $(PACKAGE_GOPATH) golang:1.14.1 make

$(BIN)-old: test
	go build -o $(BIN) -ldflags "$(LDFLAGS)"

$(BIN):
	mkdir -p ./bin
	CGO_ENABLED=0 go build -o ./bin/pf9ctl ./cmd/pf9ctl

.PHONY: generate
generate:
	./hack/update-codegen.sh
	$(MAKE) format # Ensure that the generated code is properly formatted

.PHONY: test
test: ## Run all unit and integration tests.
	go test -v ./cmd/... ./pkg/...

.PHONY: test-e2e
test-e2e: test docker-build ## Run all unit, integration, and end-to-end tests.
	E2E_DOCKER_IMAGE=$(IMAGE_NAME_TAG) ginkgo -v ./test

.PHONY: verify
verify: ## Run all static analysis checks.
	# Check if codebase is formatted.
	@which goimports > /dev/null || ! echo 'goimports not found'
	@bash -c "[ -z \"$$(goimports -l cmd pkg)\" ] && echo 'OK' || (echo 'ERROR: files are not formatted:' && goimports -l cmd pkg && echo -e \"\nRun 'make format' or manually fix the formatting issues.\n\" && false)"
	# Run static checks on codebase.
	go vet ./cmd/... ./pkg/...

.PHONY: format
format: ## Run all formatters on the codebase.
	# Format the Go codebase.
	goimports -w cmd pkg

	# Format the go.mod file.
	go mod tidy

clean:
	rm -rf $(BIN) bin
