ROX_PROJECT=apollo
TESTFLAGS=-race -p 4

.PHONY: all
all: deps

include build/protogen.mk
include build/apollo.mk
