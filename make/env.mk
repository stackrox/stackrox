# Common Makefile environment variable definitions

SHELL := /bin/bash

colon := :
comma := ,

# GOPATH might not be exported in the current shell but is available in the
# Go environment.
ifeq ($(GOPATH),)
    GOPATH=$(shell go env GOPATH)
    export GOPATH
endif

# GOPATH might actually be a colon-separated list of paths. For the purposes of
# this makefile, work with the first element only.
ifeq ($(findstring :, $(GOPATH)), $(colon))
GOPATH := $(firstword $(subst :, ,$(GOPATH)))
endif

export CGO_ENABLED ?= 1
export DEFAULT_GOOS GOARCH GOTAGS GO111MODULE GOBIN GOPROXY

# Update the arch to arm64 but only for Macs running on Apple Silicon (M1)
ifeq ($(GOARCH),)
ifeq ($(shell uname -ms),Darwin arm64)
	GOARCH := arm64
else ifeq ($(shell uname -ms),Linux aarch64)
	GOARCH := arm64
else ifeq ($(shell uname -ms),Linux ppc64le)
	GOARCH := ppc64le
else ifeq ($(shell uname -ms),Linux s390x)
	GOARCH := s390x
else
	GOARCH := amd64
endif
endif

DEFAULT_GOOS := linux
GO111MODULE := on
GOPROXY := https://proxy.golang.org|https://goproxy.io|direct

ifeq ($(GOBIN),)
GOBIN := $(GOPATH)/bin
endif

TAG := # make sure tag is never injectable as an env var
RELEASE_GOTAGS := release

# Use a release go -tag when CI is targeting a tag
ifdef CI
ifneq ($(BUILD_TAG),)
# Preserve existing GOTAGS and append release tags
GOTAGS := $(if $(GOTAGS),$(GOTAGS)$(comma))$(RELEASE_GOTAGS)
endif
endif

ifneq ($(BUILD_TAG),)
TAG := $(BUILD_TAG)
endif
