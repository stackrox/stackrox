ifeq ($(TAG),)
TAG=$(shell git describe --tags --abbrev=10 --dirty)
endif

.PHONY:
build:
	docker build -t ping-server .
	docker tag ping-server quay.io/$(REPOSITORY)/ping-server:$(TAG)
	docker push quay.io/$(REPOSITORY)/ping-server:$(TAG)
