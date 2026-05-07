#!/usr/bin/env bash

set -e

echo "Building with platform linux/${GOARCH}"
if ! docker buildx version &>/dev/null; then
    echo "Error: Docker BuildKit is required to build this image." >&2
    echo "The Dockerfile uses BuildKit features (COPY --link, syntax directive)." >&2
    echo "Please install Docker Buildx or enable BuildKit with DOCKER_BUILDKIT=1." >&2
    exit 1
fi

# SOURCE_DATE_EPOCH enables reproducible builds. For reproducible layers, we need
# rewrite-timestamp and direct registry push (BuildKit's gzip is deterministic,
# Docker daemon's is not).
if [ -n "${SOURCE_DATE_EPOCH:-}" ]; then
    echo "Using SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH for reproducible build"

    # For direct registry push, filter tags to only include registries we can push to
    # This avoids "push access denied" errors for docker.io/stackrox/* on PRs
    if [ "${DOCKER_BUILD_PUSH:-false}" = "true" ]; then
        echo "Pushing directly to registry from BuildKit for reproducible layers"

        # Extract only quay.io tags for direct push (PRs don't have docker.io creds)
        # Add architecture suffix to tags for multi-arch manifest creation
        FILTERED_ARGS=()
        while [[ $# -gt 0 ]]; do
            case "$1" in
                -t|--tag)
                    # Only include tags that start with quay.io/ and add arch suffix
                    if [[ "$2" == quay.io/* ]]; then
                        FILTERED_ARGS+=("-t" "${2}-${GOARCH}")
                    fi
                    shift 2
                    ;;
                *)
                    FILTERED_ARGS+=("$1")
                    shift
                    ;;
            esac
        done

        docker buildx build \
            --platform "linux/${GOARCH}" \
            --build-arg "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}" \
            --output "type=registry,push=true,rewrite-timestamp=true,compression=gzip" \
            "${FILTERED_ARGS[@]}"
    else
        # Load to Docker daemon (for local dev or non-PR builds that use separate push step)
        docker buildx build \
            --platform "linux/${GOARCH}" \
            --build-arg "SOURCE_DATE_EPOCH=${SOURCE_DATE_EPOCH}" \
            --output "type=docker,rewrite-timestamp=true" \
            "$@"
    fi
else
    # Local development builds without SOURCE_DATE_EPOCH
    docker buildx build --platform "linux/${GOARCH}" --load "$@"
fi
