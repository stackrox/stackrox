# Common Makefile environment variable definitions

SHELL := /bin/bash

colon := :

# GOPATH might actually be a colon-separated list of paths. For the purposes of this makefile,
# work with the first element only.

ifeq ($(findstring :, $(GOPATH)), $(colon))
GOPATH := $(patsubst :%,,$(GOPATH))
endif

export CGO_ENABLED DEFAULT_GOOS GOARCH GOTAGS GO111MODULE GOPRIVATE GOBIN
CGO_ENABLED := 1
GOARCH := amd64
DEFAULT_GOOS := linux
GO111MODULE := on
GOPRIVATE := github.com/stackrox

ifeq ($(GOBIN),)
GOBIN := $(GOPATH)/bin
endif

RELEASE_GOTAGS := release
ifdef CI
ifneq ($(CIRCLE_TAG),)
GOTAGS := $(RELEASE_GOTAGS)
TAG := $(CIRCLE_TAG)
endif
endif
