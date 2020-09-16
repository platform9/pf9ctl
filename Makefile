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

$(BIN): test
	go build -o $(BIN) -ldflags "$(LDFLAGS)"

format:
	gofmt -w -s *.go
	gofmt -w -s */*.go

clean-all: clean
	rm -rf bin

clean:
	rm -rf $(BIN)

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o pf9ctl main.go

test:
	go test -v ./...