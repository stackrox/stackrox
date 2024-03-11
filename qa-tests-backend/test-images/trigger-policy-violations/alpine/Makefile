
# Bump this when you make changes
VERSION=0.6

.PHONY: image
image:
	docker build -t trigger-policy-violations/alpine:${VERSION} .
	docker tag trigger-policy-violations/alpine:${VERSION} us.gcr.io/acs-san-stackroxci/qa/trigger-policy-violations/alpine:${VERSION}
	docker push us.gcr.io/acs-san-stackroxci/qa/trigger-policy-violations/alpine:${VERSION}
