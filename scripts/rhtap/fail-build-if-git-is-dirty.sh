#!/usr/bin/env bash

# This script works around the fact that RHTAP modifies Dockerfiles provided to it when prefetching dependencies is on.
# RHTAP changes should stop happening after https://issues.redhat.com/browse/STONEBLD-1847
# Additionally, the script returns no-zero if it detects any other changes to the git repo.
#
# If this script is not called and does not fail the build, things like `make tag` will produce `-dirty` suffix
# (as in `4.3.x-63-g09e5188ab9-dirty`) which gets embedded as the version attribute in built binaries.
#
# The script MUST be executed only from within the Dockerfile (not outside of it) because binaries are built inside.

set -euo pipefail

# When executing in RHTAP (as opposed to the script ran directly), we undo RHTAP changes to Dockerfiles.
# I found no better way to detect RHTAP than by checking the presence of cachi2.env file.
if [[ -f /cachi2/cachi2.env ]]; then
    # We can safely restore dockerfiles because the modified version of dockerfile interpreted by docker/buildah stays
    # outside, and these are local copies inside of the build context.
    git restore image/roxctl/rhtap.Dockerfile
fi

# Next, make sure no other things that make it `-dirty` slipped through. If they did, fail the build.

echo "Checking that files in git repo are not modified."
echo "If this command fails, you should see the list of modified files below."
echo "You need to find the reason and prevent that because otherwise the build results will be inconsistent."
echo ""
git status --porcelain | { ! { grep '.' >&2 && echo "ERROR: Modifies files found." >&2; } ; }

echo "No modifications git repo detected."
