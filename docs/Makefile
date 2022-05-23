.PHONY: all
all: site

.PHONY: prepare
prepare:
	# Verify that submodules have been initialized, otherwise we cannot build much.
	[ -f content/.git ] || { echo >&2 "Submodule docs/content is not initialized, please run 'git submodule update --init'" ; exit 1 ; }
	[ -f tools/.git ] || { echo >&2 "Submodule docs/tools is not initialized, please run 'git submodule update --init'" ; exit 1 ; }
	$(MAKE) -C tools prepare

.PHONY:
site: prepare
	rm -rf build/
	DOCS_INPUT=$(CURDIR)/content \
		DOCS_OUTPUT=$(CURDIR)/build \
		./tools/build-docs.sh
