ROX_PROJECT=tests

# The `-gcflags` argument is necessary to disable a new Go 1.14 runtime pointer check
# instrumentation, since not all our dependencies correctly disable instrumentation for methods using unsafe pointer
# arithmetics.
TESTFLAGS=-race -p 1 -gcflags=all=-d=checkptr=0

.PHONY: all
all: test report

include ../make/stackrox.mk
