
# Bump this when you make changes
VERSION=0.3

.PHONY: image
image:
	docker build -t trigger-policy-violations/more:${VERSION} .
	docker tag trigger-policy-violations/more:${VERSION} us.gcr.io/stackrox-ci/qa/trigger-policy-violations/more:${VERSION}
	docker push us.gcr.io/stackrox-ci/qa/trigger-policy-violations/more:${VERSION}
