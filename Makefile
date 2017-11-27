ROX_PROJECT=apollo
TESTFLAGS=-race -p 4

.PHONY: all
all: deps pkg apollo

.PHONY: pkg
pkg:
	make -C pkg

.PHONY: apollo
apollo:
	make -C apollo

include build/protogen.mk
include build/apollo.mk
