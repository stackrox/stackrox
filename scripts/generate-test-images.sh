#!/usr/bin/env bash
# Script to generate lightweight Docker images and push to quay.io/rh_ee_chsheth/image-model-test
#
# Usage: ./generate-test-images.sh [OPTIONS]
#
# Options:
#   --resume                    Resume from last successful image (reads from progress file)
#   --retry-failed              Retry only the images that failed in a previous run
#   --dry-run                   Print commands without executing them
#   --count N                   Number of images to generate (default: 2500)
#   --batch-size N              Number of images to process per batch (default: 100)
#   --sleep-between-pushes N    Seconds to sleep between pushes (default: 1)
#   --sleep-between-batches N   Seconds to sleep between batches (default: 60)
#   --max-retries N             Maximum number of retries per image (default: 3)
#   --retry-delay N             Initial delay between retries in seconds (default: 5)
#   --build-timeout N           Timeout for build operations in seconds (default: 180, i.e., 3 minutes)
#
# This script generates images by combining:
# - Lightweight base images (alpine, busybox, debian-slim, etc.)
# - Small packages (curl, wget, jq, bash, etc.)
# - Creates unique images with different combinations

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
IMAGE_GO_FILE="${REPO_ROOT}/pkg/fixtures/image.go"

# Output files (different from copy-images-to-quay.sh)
PROGRESS_FILE="${SCRIPT_DIR}/generate-images-progress.txt"
SUCCESS_LOG="${SCRIPT_DIR}/generate-images-success.log"
FAILURE_LOG="${SCRIPT_DIR}/generate-images-failure.log"
IMAGES_ENTRIES_FILE="${SCRIPT_DIR}/generated-images-entries.txt"
BUILD_DIR="${SCRIPT_DIR}/.image-build-tmp"

# Quay destination
QUAY_REPO="quay.io/rh_ee_chsheth/image-model-test"

# Default settings
IMAGE_COUNT=2500
BATCH_SIZE=100
SLEEP_BETWEEN_PUSHES=1
SLEEP_BETWEEN_BATCHES=60
MAX_RETRIES=3
RETRY_DELAY=5
BUILD_TIMEOUT=180          # 3 minutes timeout for build operations
RESUME=false
RETRY_FAILED=false
DRY_RUN=false

# Lightweight base images - all well-known, safe images
BASE_IMAGES=(
    "alpine:3.18"
    "alpine:3.19"
    "alpine:3.20"
    "alpine:latest"
    "busybox:1.36"
    "busybox:latest"
    "busybox:musl"
    "busybox:glibc"
    "debian:bookworm-slim"
    "debian:bullseye-slim"
    "ubuntu:22.04"
    "ubuntu:24.04"
    "python:3.10-alpine"
    "python:3.11-alpine"
    "python:3.12-alpine"
    "golang:1.21-alpine"
    "golang:1.22-alpine"
    "node:20-alpine"
    "node:22-alpine"
    "ruby:3.2-alpine"
    "ruby:3.3-alpine"
    "rust:alpine"
    "redis:alpine"
    "nginx:alpine"
    "postgres:alpine"
    "memcached:alpine"
    "traefik:latest"
    "haproxy:alpine"
    "httpd:alpine"
    "caddy:alpine"
)

# Packages/tools to add (for alpine-based images)
ALPINE_PACKAGES=(
    "curl"
    "wget"
    "jq"
    "bash"
    "git"
    "openssh-client"
    "ca-certificates"
    "openssl"
    "netcat-openbsd"
    "bind-tools"
    "iputils"
    "tcpdump"
    "strace"
    "htop"
    "vim"
    "nano"
    "less"
    "tree"
    "file"
    "gzip"
    "tar"
    "unzip"
    "zip"
    "make"
    "gcc"
    "musl-dev"
    "python3"
    "py3-pip"
    "nodejs"
    "npm"
)

# Packages for debian/ubuntu-based images
DEBIAN_PACKAGES=(
    "curl"
    "wget"
    "jq"
    "git"
    "openssh-client"
    "ca-certificates"
    "openssl"
    "netcat-openbsd"
    "dnsutils"
    "iputils-ping"
    "tcpdump"
    "strace"
    "htop"
    "vim"
    "nano"
    "less"
    "tree"
    "file"
    "gzip"
    "tar"
    "unzip"
    "zip"
    "make"
    "gcc"
)

# Simple commands/labels to add variety
LABELS=(
    "version=1.0"
    "version=2.0"
    "version=3.0"
    "env=dev"
    "env=test"
    "env=staging"
    "env=prod"
    "tier=frontend"
    "tier=backend"
    "tier=database"
    "tier=cache"
    "tier=proxy"
)

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --resume)
            RESUME=true
            shift
            ;;
        --retry-failed)
            RETRY_FAILED=true
            shift
            ;;
        --dry-run)
            DRY_RUN=true
            shift
            ;;
        --count)
            IMAGE_COUNT="$2"
            shift 2
            ;;
        --batch-size)
            BATCH_SIZE="$2"
            shift 2
            ;;
        --sleep-between-pushes)
            SLEEP_BETWEEN_PUSHES="$2"
            shift 2
            ;;
        --sleep-between-batches)
            SLEEP_BETWEEN_BATCHES="$2"
            shift 2
            ;;
        --max-retries)
            MAX_RETRIES="$2"
            shift 2
            ;;
        --retry-delay)
            RETRY_DELAY="$2"
            shift 2
            ;;
        --build-timeout)
            BUILD_TIMEOUT="$2"
            shift 2
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

echo "=== Image Generation Script ==="
echo "Settings:"
echo "  Target image count: ${IMAGE_COUNT}"
echo "  Batch size: ${BATCH_SIZE}"
echo "  Sleep between pushes: ${SLEEP_BETWEEN_PUSHES}s"
echo "  Sleep between batches: ${SLEEP_BETWEEN_BATCHES}s"
echo "  Max retries: ${MAX_RETRIES}"
echo "  Initial retry delay: ${RETRY_DELAY}s"
echo "  Build timeout: ${BUILD_TIMEOUT}s"
echo "  Resume mode: ${RESUME}"
echo "  Retry failed: ${RETRY_FAILED}"
echo "  Dry run: ${DRY_RUN}"
echo ""

# Check if logged into docker and quay
if [[ "${DRY_RUN}" == "false" ]]; then
    echo "Checking Docker status..."
    if ! docker info >/dev/null 2>&1; then
        echo "ERROR: Docker is not running or not accessible"
        exit 1
    fi
    
    echo "NOTE: Make sure you are logged into Quay.io:"
    echo "  docker login quay.io"
    echo ""
    read -p "Press Enter to continue or Ctrl+C to abort..."
    
    # Create build directory
    mkdir -p "${BUILD_DIR}"
fi

# Timeout wrapper function - runs a command with a timeout
# Usage: run_with_timeout <timeout_seconds> command [args...]
# Returns: 0 on success, 124 on timeout, or command's exit code
run_with_timeout() {
    local timeout_secs=$1
    shift
    
    local os_type
    os_type=$(uname -s)
    
    if [[ "${os_type}" == "Linux" ]]; then
        # Linux: use the timeout command from coreutils
        # --foreground: needed for proper signal handling in scripts
        # --kill-after=5: send SIGKILL 5s after initial signal if still running
        timeout --foreground --kill-after=5 --signal=TERM "${timeout_secs}" "$@"
        return $?
    elif [[ "${os_type}" == "Darwin" ]]; then
        # macOS: try gtimeout (from coreutils via Homebrew), otherwise use Perl
        # --foreground: needed for proper signal handling in scripts
        # --kill-after=5: send SIGKILL 5s after initial signal if still running
        if [[ -x "/opt/homebrew/bin/gtimeout" ]]; then
            /opt/homebrew/bin/gtimeout --foreground --kill-after=5 --signal=TERM "${timeout_secs}" "$@"
            return $?
        elif [[ -x "/usr/local/bin/gtimeout" ]]; then
            /usr/local/bin/gtimeout --foreground --kill-after=5 --signal=TERM "${timeout_secs}" "$@"
            return $?
        else
            # Use Perl for timeout on macOS (Perl is always available)
            # This properly handles signals and kills the process group
            perl -e '
                use strict;
                use warnings;
                use POSIX qw(setsid);
                
                my $timeout = shift @ARGV;
                my $pid = fork();
                
                if (!defined $pid) {
                    die "Fork failed: $!";
                } elsif ($pid == 0) {
                    # Child: create new process group and exec
                    setsid();
                    exec(@ARGV) or die "Exec failed: $!";
                } else {
                    # Parent: wait with timeout
                    eval {
                        local $SIG{ALRM} = sub { die "timeout\n" };
                        alarm($timeout);
                        waitpid($pid, 0);
                        alarm(0);
                    };
                    if ($@ && $@ eq "timeout\n") {
                        # Timeout occurred - kill the process group
                        kill("-KILL", $pid);  # Kill entire process group
                        waitpid($pid, 0);
                        exit(124);  # Timeout exit code
                    }
                    # Return child exit status
                    exit($? >> 8);
                }
            ' "${timeout_secs}" "$@"
            return $?
        fi
    else
        # Unknown OS: try perl-based timeout
        perl -e '
            my $timeout = shift @ARGV;
            my $pid = fork();
            if ($pid == 0) {
                exec(@ARGV);
            } else {
                eval {
                    local $SIG{ALRM} = sub { die "timeout\n" };
                    alarm($timeout);
                    waitpid($pid, 0);
                    alarm(0);
                };
                if ($@) {
                    kill(9, $pid);
                    exit(124);
                }
                exit($? >> 8);
            }
        ' "${timeout_secs}" "$@"
        return $?
    fi
}

# Retry wrapper function with exponential backoff
retry_with_backoff() {
    local description=$1
    shift
    local attempt=1
    local delay=${RETRY_DELAY}
    
    while [[ ${attempt} -le ${MAX_RETRIES} ]]; do
        if "$@" 2>&1; then
            return 0
        fi
        
        if [[ ${attempt} -lt ${MAX_RETRIES} ]]; then
            echo "    ${description} failed (attempt ${attempt}/${MAX_RETRIES}). Retrying in ${delay}s..."
            sleep "${delay}"
            delay=$((delay * 2))
        fi
        
        ((attempt++))
    done
    
    echo "    ${description} failed after ${MAX_RETRIES} attempts"
    return 1
}

# Get start index for resume
get_start_index() {
    if [[ "${RESUME}" == "true" && -f "${PROGRESS_FILE}" ]]; then
        cat "${PROGRESS_FILE}"
    else
        echo "0"
    fi
}

# Save progress
save_progress() {
    local index=$1
    echo "${index}" > "${PROGRESS_FILE}"
}

# Generate a unique image name based on index
generate_image_config() {
    local index=$1
    
    local base_count=${#BASE_IMAGES[@]}
    local pkg_count=${#ALPINE_PACKAGES[@]}
    local label_count=${#LABELS[@]}
    
    # Calculate indices for each component
    local base_idx=$((index % base_count))
    local pkg_idx=$(( (index / base_count) % pkg_count ))
    local label_idx=$(( (index / (base_count * pkg_count)) % label_count ))
    # Use label_idx as variant since label differentiates images with same base+package
    # This ensures unique names for each base+package+label combination
    local variant=${label_idx}
    
    local base_image="${BASE_IMAGES[$base_idx]}"
    local label="${LABELS[$label_idx]}"
    
    # Determine package based on base image type
    local package
    if [[ "${base_image}" == *"alpine"* ]] || [[ "${base_image}" == *"busybox"* ]]; then
        package="${ALPINE_PACKAGES[$pkg_idx]}"
    else
        package="${DEBIAN_PACKAGES[$((pkg_idx % ${#DEBIAN_PACKAGES[@]}))]}"
    fi
    
    # Create a unique image name
    # Format: img-<base>-<package>-v<variant> where variant corresponds to label_idx
    local safe_base="${base_image//[:\/]/-}"
    local image_name="img-${safe_base}-${package}-v${variant}"
    
    echo "${base_image}|${package}|${label}|${image_name}"
}

# Check if base image is alpine-based
is_alpine_based() {
    local base_image=$1
    if [[ "${base_image}" == *"alpine"* ]] || \
       [[ "${base_image}" == "busybox"* ]] || \
       [[ "${base_image}" == "redis:alpine"* ]] || \
       [[ "${base_image}" == "nginx:alpine"* ]] || \
       [[ "${base_image}" == "postgres:alpine"* ]] || \
       [[ "${base_image}" == "memcached:alpine"* ]] || \
       [[ "${base_image}" == "haproxy:alpine"* ]] || \
       [[ "${base_image}" == "httpd:alpine"* ]] || \
       [[ "${base_image}" == "caddy:alpine"* ]]; then
        return 0
    fi
    return 1
}

# Generate Dockerfile content
generate_dockerfile() {
    local base_image=$1
    local package=$2
    local label=$3
    local image_name=$4
    
    local dockerfile=""
    
    # For busybox, we can't install packages, just add labels and a simple command
    if [[ "${base_image}" == "busybox"* ]]; then
        dockerfile="FROM ${base_image}
LABEL ${label}
LABEL generator=stackrox-test
LABEL image-name=${image_name}
RUN echo 'Generated test image: ${image_name}' > /info.txt
CMD [\"cat\", \"/info.txt\"]
"
    elif is_alpine_based "${base_image}"; then
        dockerfile="FROM ${base_image}
LABEL ${label}
LABEL generator=stackrox-test
LABEL image-name=${image_name}
RUN apk add --no-cache ${package} || true
RUN echo 'Generated test image: ${image_name}' > /info.txt
CMD [\"cat\", \"/info.txt\"]
"
    else
        # Debian/Ubuntu based
        dockerfile="FROM ${base_image}
LABEL ${label}
LABEL generator=stackrox-test
LABEL image-name=${image_name}
RUN apt-get update && apt-get install -y --no-install-recommends ${package} || true && rm -rf /var/lib/apt/lists/*
RUN echo 'Generated test image: ${image_name}' > /info.txt
CMD [\"cat\", \"/info.txt\"]
"
    fi
    
    echo "${dockerfile}"
}

# Build and push a single image
process_image() {
    local index=$1
    local total=$2
    
    # Generate image configuration
    local config
    config=$(generate_image_config "${index}")
    
    IFS='|' read -r base_image package label image_name <<< "${config}"
    
    local orig_tag="${QUAY_REPO}:${image_name}_orig"
    local copy_tag="${QUAY_REPO}:${image_name}_copy"
    
    echo "[${index}/${total}] Processing: ${image_name}"
    echo "  Base: ${base_image}, Package: ${package}, Label: ${label}"
    echo "  Tags: ${orig_tag}"
    echo "        ${copy_tag}"
    
    if [[ "${DRY_RUN}" == "true" ]]; then
        echo "  [DRY RUN] Would build and push image"
        # Add entries to the file for dry run
        echo "{\"${orig_tag}\", \"sha256:placeholder_${index}_orig\"}," >> "${IMAGES_ENTRIES_FILE}"
        echo "{\"${copy_tag}\", \"sha256:placeholder_${index}_copy\"}," >> "${IMAGES_ENTRIES_FILE}"
        return 0
    fi
    
    # Generate Dockerfile
    local dockerfile_path="${BUILD_DIR}/Dockerfile.${index}"
    generate_dockerfile "${base_image}" "${package}" "${label}" "${image_name}" > "${dockerfile_path}"
    
    # Build image with timeout
    echo "  Building (timeout: ${BUILD_TIMEOUT}s)..."
    local build_exit_code
    run_with_timeout "${BUILD_TIMEOUT}" docker build -t "${orig_tag}" -f "${dockerfile_path}" "${BUILD_DIR}" 2>&1
    build_exit_code=$?
    
    if [[ ${build_exit_code} -eq 124 ]]; then
        echo "  FAILED: Build timed out after ${BUILD_TIMEOUT}s - skipping"
        echo "${image_name},build_timeout,${base_image},${package}" >> "${FAILURE_LOG}"
        rm -f "${dockerfile_path}"
        # Clean up any partial images
        docker rmi "${orig_tag}" 2>/dev/null || true
        return 1
    elif [[ ${build_exit_code} -ne 0 ]]; then
        echo "  FAILED: Build failed (exit code: ${build_exit_code})"
        echo "${image_name},build_failed,${base_image},${package}" >> "${FAILURE_LOG}"
        rm -f "${dockerfile_path}"
        return 1
    fi
    
    # Tag as copy
    echo "  Tagging copy..."
    if ! docker tag "${orig_tag}" "${copy_tag}" 2>&1; then
        echo "  FAILED: Tag failed"
        echo "${image_name},tag_failed,${base_image},${package}" >> "${FAILURE_LOG}"
        docker rmi "${orig_tag}" 2>/dev/null || true
        rm -f "${dockerfile_path}"
        return 1
    fi
    
    # Push orig
    echo "  Pushing orig..."
    if ! retry_with_backoff "Push orig" docker push "${orig_tag}"; then
        echo "  FAILED: Push orig failed"
        echo "${image_name},push_orig_failed,${base_image},${package}" >> "${FAILURE_LOG}"
        docker rmi "${orig_tag}" "${copy_tag}" 2>/dev/null || true
        rm -f "${dockerfile_path}"
        return 1
    fi
    
    # Get digest for orig
    local orig_digest
    orig_digest=$(docker inspect --format='{{index .RepoDigests 0}}' "${orig_tag}" 2>/dev/null | sed 's/.*@//')
    
    # Push copy
    echo "  Pushing copy..."
    if ! retry_with_backoff "Push copy" docker push "${copy_tag}"; then
        echo "  FAILED: Push copy failed"
        echo "${image_name},push_copy_failed,${base_image},${package}" >> "${FAILURE_LOG}"
        docker rmi "${orig_tag}" "${copy_tag}" 2>/dev/null || true
        rm -f "${dockerfile_path}"
        return 1
    fi
    
    # Get digest for copy (should be same as orig since it's the same image)
    local copy_digest
    copy_digest=$(docker inspect --format='{{index .RepoDigests 0}}' "${copy_tag}" 2>/dev/null | sed 's/.*@//')
    
    # Cleanup
    echo "  Cleaning up..."
    docker rmi "${orig_tag}" "${copy_tag}" 2>/dev/null || true
    rm -f "${dockerfile_path}"
    
    # Log success
    echo "${orig_tag},${orig_digest}" >> "${SUCCESS_LOG}"
    echo "${copy_tag},${copy_digest}" >> "${SUCCESS_LOG}"
    
    # Add to entries file
    if [[ -n "${orig_digest}" ]]; then
        echo "{\"${orig_tag}\", \"${orig_digest}\"}," >> "${IMAGES_ENTRIES_FILE}"
    else
        echo "{\"${orig_tag}\", \"sha256:unknown\"}," >> "${IMAGES_ENTRIES_FILE}"
    fi
    if [[ -n "${copy_digest}" ]]; then
        echo "{\"${copy_tag}\", \"${copy_digest}\"}," >> "${IMAGES_ENTRIES_FILE}"
    else
        echo "{\"${copy_tag}\", \"sha256:unknown\"}," >> "${IMAGES_ENTRIES_FILE}"
    fi
    
    echo "  SUCCESS"
    return 0
}

# Extract failed images from the failure log
extract_failed_images() {
    if [[ ! -f "${FAILURE_LOG}" ]]; then
        echo ""
        return
    fi
    # Format in failure log: image_name,failure_reason,base_image,package
    # We just need the index which we can derive from the image name
    awk -F',' '{print NR-1}' "${FAILURE_LOG}"
}

# Main processing loop
process_all_images() {
    echo "Generating ${IMAGE_COUNT} images..."
    
    local start_index
    start_index=$(get_start_index)
    echo "Starting from index: ${start_index}"
    
    # Initialize log files if not resuming
    if [[ "${start_index}" == "0" ]]; then
        : > "${SUCCESS_LOG}"
        : > "${FAILURE_LOG}"
        : > "${IMAGES_ENTRIES_FILE}"
    fi
    
    local batch_count=0
    local success_count=0
    local failure_count=0
    
    for ((i=start_index; i<IMAGE_COUNT; i++)); do
        # Process the image
        if process_image "$((i+1))" "${IMAGE_COUNT}"; then
            ((success_count++)) || true
        else
            ((failure_count++)) || true
        fi
        
        # Save progress
        save_progress "$((i+1))"
        
        # Batch management
        ((batch_count++)) || true
        
        if [[ "${batch_count}" -ge "${BATCH_SIZE}" && "$((i+1))" -lt "${IMAGE_COUNT}" ]]; then
            batch_count=0
            echo ""
            echo "=== Batch complete. Sleeping for ${SLEEP_BETWEEN_BATCHES}s ==="
            echo "Progress: $((i+1))/${IMAGE_COUNT} (Success: ${success_count}, Failures: ${failure_count})"
            echo ""
            
            if [[ "${DRY_RUN}" == "false" ]]; then
                sleep "${SLEEP_BETWEEN_BATCHES}"
            fi
        elif [[ "${DRY_RUN}" == "false" ]]; then
            sleep "${SLEEP_BETWEEN_PUSHES}"
        fi
    done
    
    echo ""
    echo "=== Processing Complete ==="
    echo "Total processed: ${IMAGE_COUNT}"
    echo "Successes: ${success_count}"
    echo "Failures: ${failure_count}"
    
    print_summary
}

# Retry failed images
retry_failed_images() {
    echo "Extracting failed images from ${FAILURE_LOG}..."
    
    if [[ ! -f "${FAILURE_LOG}" ]]; then
        echo "No failure log found. Nothing to retry."
        exit 0
    fi
    
    # Read failed image configs
    mapfile -t failed_configs < "${FAILURE_LOG}"
    local total=${#failed_configs[@]}
    
    if [[ ${total} -eq 0 ]]; then
        echo "No failed images to retry."
        exit 0
    fi
    
    echo "Found ${total} failed images to retry"
    
    # Backup the old failure log
    local old_failure_log="${FAILURE_LOG}.bak.$(date +%Y%m%d_%H%M%S)"
    mv "${FAILURE_LOG}" "${old_failure_log}"
    echo "Backed up old failure log to: ${old_failure_log}"
    : > "${FAILURE_LOG}"
    
    local batch_count=0
    local success_count=0
    local failure_count=0
    
    for ((i=0; i<total; i++)); do
        IFS=',' read -r image_name _ base_image package <<< "${failed_configs[$i]}"
        
        # Re-derive the index from the image name pattern
        # This is a simplified retry - we process by reconstructing
        if process_image "$((i+1))" "${total}"; then
            ((success_count++)) || true
        else
            ((failure_count++)) || true
        fi
        
        # Batch management
        ((batch_count++)) || true
        
        if [[ "${batch_count}" -ge "${BATCH_SIZE}" && "$((i+1))" -lt "${total}" ]]; then
            batch_count=0
            echo ""
            echo "=== Batch complete. Sleeping for ${SLEEP_BETWEEN_BATCHES}s ==="
            echo "Progress: $((i+1))/${total} (Success: ${success_count}, Failures: ${failure_count})"
            echo ""
            
            if [[ "${DRY_RUN}" == "false" ]]; then
                sleep "${SLEEP_BETWEEN_BATCHES}"
            fi
        elif [[ "${DRY_RUN}" == "false" ]]; then
            sleep "${SLEEP_BETWEEN_PUSHES}"
        fi
    done
    
    echo ""
    echo "=== Retry Complete ==="
    echo "Total retried: ${total}"
    echo "Successes: ${success_count}"
    echo "Failures: ${failure_count}"
    
    if [[ ${failure_count} -gt 0 ]]; then
        echo ""
        echo "Some images still failed. Run with --retry-failed again to retry them."
    fi
    
    print_summary
}

# Print summary
print_summary() {
    echo ""
    echo "Logs saved to:"
    echo "  Success: ${SUCCESS_LOG}"
    echo "  Failures: ${FAILURE_LOG}"
    echo "  Image entries: ${IMAGES_ENTRIES_FILE}"
    
    if [[ -f "${FAILURE_LOG}" && -s "${FAILURE_LOG}" ]]; then
        local fail_count
        fail_count=$(wc -l < "${FAILURE_LOG}" | tr -d ' ')
        echo ""
        echo "NOTE: ${fail_count} images failed. To retry them, run:"
        echo "  ${SCRIPT_DIR}/generate-test-images.sh --retry-failed"
    fi
    
    echo ""
    echo "To add the ImageCopies list to the Go file, run:"
    echo "  ${SCRIPT_DIR}/insert-generated-images.sh"
    
    # Cleanup build directory
    if [[ -d "${BUILD_DIR}" ]]; then
        rm -rf "${BUILD_DIR}"
    fi
}

# Main entry point
main() {
    if [[ "${RETRY_FAILED}" == "true" ]]; then
        retry_failed_images
    else
        process_all_images
    fi
}

main
