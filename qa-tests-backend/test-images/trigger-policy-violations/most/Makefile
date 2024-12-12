
# Bump this when you make changes
VERSION=0.20

.PHONY: image
image:
	docker build --platform linux/amd64 -t trigger-policy-violations/most:${VERSION} .
	docker tag trigger-policy-violations/most:${VERSION} us.gcr.io/acs-san-stackroxci/qa/trigger-policy-violations/most:${VERSION}
	docker push us.gcr.io/acs-san-stackroxci/qa/trigger-policy-violations/most:${VERSION}
