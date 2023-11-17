.DEFAULT_GOAL = all

PERF_SCALE_QA_OPEN_CLOSE_PORTS_LOAD_TAG := perf-scale-open-close-ports-load

ifneq ($(PERF_SCALE_QA_TAG),)
PERF_SCALE_QA_OPEN_CLOSE_PORTS_LOAD_TAG=perf-scale-open-close-ports-load-$(PERF_SCALE_QA_TAG)
endif

.PHONY: all
all:
	@docker build -t quay.io/rhacs-eng/qa:$(PERF_SCALE_QA_OPEN_CLOSE_PORTS_LOAD_TAG) .

.PHONY: push
push:
	@docker push quay.io/rhacs-eng/qa:$(PERF_SCALE_QA_OPEN_CLOSE_PORTS_LOAD_TAG)
