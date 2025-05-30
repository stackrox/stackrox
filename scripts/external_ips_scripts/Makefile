ifeq ($(TAG),)
TAG=$(shell git describe --tags --abbrev=10 --dirty)
endif

.PHONY:
build:
	docker build -t external-connection .
	docker tag external-connection quay.io/$(REPOSITORY)/external-connection:$(TAG)
	docker push quay.io/$(REPOSITORY)/external-connection:$(TAG)
