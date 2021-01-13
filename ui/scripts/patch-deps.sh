#!/usr/bin/env bash

#####
# This script patches UI dependencies when issues are still not fixed upstream.
# Should be invoked from the `ui` directory (parent for this `scripts/` dir).
#
# Currently the following dependencies are patched
#   - react-dev-utils/ModuleScopePlugin - react-scripts has an issue with importing CSS modules from CSS files
#     when it happens in the context of yarn workspaces monorepo.
#     See https://github.com/facebook/create-react-app/issues/10373
#####

find ./ -path "*/react-dev-utils/ModuleScopePlugin.js" -exec patch --forward {} ./scripts/ModuleScopePlugin.js.patch \; 
