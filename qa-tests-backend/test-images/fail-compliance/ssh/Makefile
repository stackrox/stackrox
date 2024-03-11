
# Bump this when you make changes
VERSION=0.1

.PHONY: image
image:
	docker build -t fail-compliance/ssh:${VERSION} .
	docker tag fail-compliance/ssh:${VERSION} us.gcr.io/acs-san-stackroxci/qa/fail-compliance/ssh:${VERSION}
	docker push us.gcr.io/acs-san-stackroxci/qa/fail-compliance/ssh:${VERSION}
