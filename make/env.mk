# Common Makefile environment variable definitions

SHELL := /bin/bash

colon := :

# GOPATH might actually be a colon-separated list of paths. For the purposes of this makefile,
# work with the first element only.

ifeq ($(findstring :, $(GOPATH)), $(colon))
GOPATH := $(firstword $(subst :, ,$(GOPATH)))
endif

export CGO_ENABLED DEFAULT_GOOS GOARCH GOTAGS GO111MODULE GOPRIVATE GOBIN GOPROXY
CGO_ENABLED := 1

# Update the arch to arm64 but only for Macs running on Apple Silicon (M1)
ifeq ($(shell uname -ms),Darwin arm64)
	GOARCH := arm64
else ifeq ($(shell uname -ms),Linux aarch64)
	GOARCH := arm64
else
	GOARCH := amd64
endif

DEFAULT_GOOS := linux
GO111MODULE := on
GOPRIVATE := github.com/stackrox
GOPROXY := https://proxy.golang.org|https://goproxy.io|direct

ifeq ($(GOBIN),)
GOBIN := $(GOPATH)/bin
endif

TAG := # make sure tag is never injectable as an env var
RELEASE_GOTAGS := release
ifdef CI
ifneq ($(CIRCLE_TAG),)
GOTAGS := $(RELEASE_GOTAGS)
TAG := $(CIRCLE_TAG)
endif
endif
