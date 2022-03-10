# ROX_CI_IMAGE := quay.io/cgorman1/apollo-ci:0.3.18
ROX_CI_IMAGE   := rox-ci-image:latest

lint:
	shellcheck --external-sources --source-path=./scripts ./scripts/*.sh


# TODO(sbostick): support running locally on macos?
# TODO(sbostick): support running locally in ROX_CI_IMAGE?
#

#pull-rox-ci-image:
#	pbpaste | docker login quay.io --username "sbostick@redhat.com" --password-stdin
#	docker pull ${ROX_CI_IMAGE}  # Unauthorized -- have source code so building locally instead
#
#build-rox-ci-image:
#	#cd ~/source/rox-ci-image && docker build images/ -f ./images/Dockerfile.rox -t ${ROX_CI_IMAGE}
#	cd ~/source/rox-ci-image && docker build -f ./images/Dockerfile.rox -t ${ROX_CI_IMAGE}
#	docker images ${ROX_CI_IMAGE}
#
#run-qa-tests:
#	@rm -rf "${LOCAL_DATA_DIR}"
#	@mkdir -p "${LOCAL_DATA_DIR}"
#	@docker run -it --rm --name "run-qa-tests" --entrypoint "" \
#		-e REGISTRY_USERNAME="$(pass quay-io-username)" \
#		-e REGISTRY_PASSWORD="$(pass quay-io-password)" \
#		-v "${LOCAL_DATA_DIR}:/data" \
#		-v "$$HOME/go/src/github.com/stackrox/rox:/source/rox" \
#		-v "$$HOME/source/workflow:/source/workflow:ro" \
#		"$(ROX_CI_IMAGE)" \
#		/bin/bash --login
