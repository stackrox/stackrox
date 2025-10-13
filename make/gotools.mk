# gotools.mk
# Simplified installation & usage of Go-based tools
#
# Input variables:
#   GOTOOLS_PROJECT_ROOT: the project root directory; defaults to $(CURDIR)
#   GOTOOLS_ROOT: the directory in which this file stores auxiliary data (should be .gitignore'd); defaults to
#                 $(GOTOOLS_PROJECT_ROOT)/.gotools
#   GOTOOLS_BIN: the directory in which binaries are stored; defaults to $(GOTOOLS_ROOT)/bin.
#
# This file defines a single (user-facing) macro, `go-tool`, which can be invoked via
#   $(call go-tool VARNAME, go-pkg [, module-root])
# where go-pkg can be:
# - an absolute Go import path with an explicit version, e.g.,
#     github.com/golangci/golangci-lint/cmd/golangci-lint@v1.49.0. In this case, the tool is installed via `go install`,
#     and module information from the local workspace is ignored, in accordance with the normal behavior of go install
#     with an explicit version given.
# - an absolute Go import path WITHOUT a version, e.g., github.com/golangci/golangci-lint/cmd/golangci-lint. In this
#     case, the tool is installed via `go install` from the module rooted at $(GOTOOLS_PROJECT_ROOT), or, if
#     module-root is given, from the module rooted at that (relative) path. I.e., go-pkg must be provided by a module
#     listed as a requirement in <module-root>/go.mod.
# - a relative Go import path (WITHOUT a version), e.g., ./tools/roxvet. In this case, the tool is installed via
#     `go install` from the module rooted at $(GOTOOLS_PROJECT_ROOT).
#
# Invoking go-tool will set up Makefile rules to build the tools, using reasonable strategies for caching to avoid
# building a tool multiple times. In particular:
# - when using an absolute Go import path with a version, the rule is set up such that the `go install` command is only
#   run once.
# - when using an absolute Go import path without a version, the rule is set up such that the `go install` command is
#   re-run only when the respective go.mod file changes.
# - when using a relative Go import path, the rule is set up such that the `go install` command is re-run on every
#   `make` invocation.
# Note that `go install` uses a pretty effective caching strategy under the hood, so even with relative import path,
# you should not expect noticeable latency.
#
# In addition to setting up the rules for building, invoking go-tool will also set the value of the variable `VARNAME`
# to the (canonical) location of the respective tool's binary, which is $(GOTOOLS_BIN)/<binary basename>. `$(VARNAME)`
# should be used as the only way of both invoking the tool in the Makefile as well as expressing a dependency on the
# installation of the tool.
# For use in non-Makefile scripts, a target `which-<tool>` is added, whhere <tool> is the basename of the tool binary.
# This target prints the canonical location of the binary and, if necessary, builds it. Note that invocations of
# `make which-tool` should be made with the flags `--quiet --no-print-directory` set, as otherwise the output gets
# clobbered.
#
# This file also defines two static, global targets:
#   gotools-clean: this removes all gotools-related data
#   gotools-all: this builds all gotools.

GOTOOLS_PROJECT_ROOT ?= $(CURDIR)
GOTOOLS_ROOT ?= $(GOTOOLS_PROJECT_ROOT)/.gotools
GOTOOLS_BIN ?= $(GOTOOLS_ROOT)/bin

_GOTOOLS_ALL_GOTOOLS :=

define go-tool-impl
# The variable via which the executable can be referenced
_gotools_var_name := $(strip $(1))
# The importable Go package path that contains the "main" package for the tool
_gotools_pkg := $(firstword $(subst @, ,$(strip $(2))))
# The version of the tool (if a version was explicitly specified)
_gotools_version := $(word 2,$(subst @, ,$(strip $(2))))
# The folder containing the go.mod file, if not the root folder
ifeq ($(strip $(3)),)
_gotools_mod_root := $(GOTOOLS_PROJECT_ROOT)
else
_gotools_mod_root := $(strip $(3))
endif

# We need to strip a `/v2` (etc.) suffix to derive the tool binary's basename.
_gotools_bin_name := $$(notdir $$(shell echo "$$(_gotools_pkg)" | sed -E 's@/v[[:digit:]]+$$$$@@g'))
_gotools_canonical_bin_path := $(GOTOOLS_BIN)/$$(_gotools_bin_name)
$$(_gotools_var_name) := $$(_gotools_canonical_bin_path)

.PHONY: which-$$(_gotools_bin_name)
which-$$(_gotools_bin_name):
	@$(MAKE) $$($(strip $(1))) >&2
	@echo $$($(strip $(1)))

ifneq ($(filter ./%,$(2)),)
# Tool is built from local files. We have to introduce a phony target and let the Go compiler
# do all the caching.
.PHONY: $$(_gotools_canonical_bin_path)
$$(_gotools_canonical_bin_path):
	@echo "+ $$(notdir $$@)"
	$$(SILENT)GOBIN="$$(dir $$@)" go install "$(strip $(2))"
else
# Tool is specified with version, so we don't take any info from the go.mod file.
# We install the tool into a location that is version-dependent, and build it via this target. Since the name of
# the tool under that path is version-dependent, we never have to rebuild it, as it's either the correct version, or
# does not exist.
ifneq ($$(_gotools_version),)
_gotools_versioned_bin_path := $(GOTOOLS_ROOT)/versioned/$$(_gotools_pkg)/$$(_gotools_version)/$$(_gotools_bin_name)
$$(_gotools_versioned_bin_path):
	@echo "+ $$(notdir $$@)"
	$$(SILENT)GOBIN="$$(dir $$@)" go install "$(strip $(2))"

# To make the tool accessible in the canonical location, we create a symlink. This only depends on the versioned path,
# i.e., only needs to be recreated when the version is bumped.
$$(_gotools_canonical_bin_path): $$(_gotools_versioned_bin_path)
	@mkdir -p "$(GOTOOLS_BIN)"
	$$(SILENT)ln -sf "$$<" "$$@"

else

# Tool is specified with an absolute path without a version. Take info from go.mod file in the respective directory.
$$(_gotools_canonical_bin_path): $$(_gotools_mod_root)/go.mod $$(_gotools_mod_root)/go.sum
	@echo "+ $$(notdir $$@)"
	$$(SILENT)cd "$$(dir $$<)" && GOBIN="$$(dir $$@)" go install "$(strip $(2))"

endif
endif

_GOTOOLS_ALL_GOTOOLS += $$(_gotools_canonical_bin_path)

endef

go-tool = $(eval $(call go-tool-impl,$(1),$(2),$(3)))


.PHONY: gotools-clean
gotools-clean:
	@echo "+ $@"
	@git clean -dfX "$(GOTOOLS_ROOT)"  # don't use rm -rf to avoid catastrophes

.PHONY: gotools-all
gotools-all:
	@# these cannot be dependencies, as we need `$(_GOTOOLS_ALL_GOTOOLS)` to be
	@# evaluated when the target is actually run.
	$(MAKE) $(_GOTOOLS_ALL_GOTOOLS)
