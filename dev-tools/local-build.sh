#!/usr/bin/env bash
set -euo pipefail

# Fast local StackRox build using Tekton
# Usage: ./dev-tools/local-build.sh [options]

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default configuration
REGISTRY="${ROX_LOCAL_REGISTRY:-localhost:5000}"
TAG="${ROX_LOCAL_TAG:-latest}"
NAMESPACE="${TEKTON_NAMESPACE:-stackrox-builds}"
REPO_URL="${REPO_URL:-$(git remote get-url origin 2>/dev/null || echo "https://github.com/stackrox/stackrox.git")}"
REVISION="${REVISION:-$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "master")}"
BUILDER_IMAGE="${BUILDER_IMAGE:-quay.io/stackrox-io/apollo-ci:stackrox-build-0.4.9}"
CACHE_BUCKET="${CACHE_BUCKET:-local-dev-cache}"
MINIO_HOST="${MINIO_HOST:-minio.default.svc:9000}"

# Script options
SETUP_MINIO=true
SETUP_TEKTON=true
WAIT_FOR_COMPLETION=true
DEPLOY_STACKROX=false
VERBOSE=false

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

log() {
    echo -e "${BLUE}[$(date +'%H:%M:%S')]${NC} $*"
}

warn() {
    echo -e "${YELLOW}[WARNING]${NC} $*"
}

error() {
    echo -e "${RED}[ERROR]${NC} $*"
}

success() {
    echo -e "${GREEN}[SUCCESS]${NC} $*"
}

usage() {
    cat << EOF
Fast StackRox Tekton Build Tool

USAGE:
    $0 [OPTIONS]

OPTIONS:
    -r, --registry REGISTRY     Container registry (default: localhost:5000)
    -t, --tag TAG              Image tag (default: latest)
    -n, --namespace NAMESPACE   Tekton namespace (default: stackrox-builds)
    --repo-url URL             Git repository URL (default: current origin)
    --revision REV             Git revision (default: current branch)
    --builder-image IMAGE      Builder image (default: official StackRox)
    --cache-bucket BUCKET      S3 cache bucket (default: local-dev-cache)
    --minio-host HOST          MinIO host (default: minio.default.svc:9000)

    --no-setup-minio          Skip MinIO setup
    --no-setup-tekton         Skip Tekton resource setup
    --no-wait                 Don't wait for pipeline completion
    --deploy                  Deploy StackRox after successful build
    -v, --verbose             Verbose output
    -h, --help                Show this help

ENVIRONMENT VARIABLES:
    ROX_LOCAL_REGISTRY        Default registry (overrides --registry)
    ROX_LOCAL_TAG             Default tag (overrides --tag)
    TEKTON_NAMESPACE          Default Tekton namespace
    ROX_IMAGE_FLAVOR          Set to 'local-dev' for automatic Helm integration

EXAMPLES:
    # Basic local build
    $0

    # Custom registry and tag
    $0 --registry my-registry:5000 --tag v1.0.0

    # Build and deploy
    $0 --deploy

    # Build from specific branch
    $0 --revision feature-branch

    # Use custom builder image
    $0 --builder-image quay.io/my-org/builder:latest
EOF
}

check_prerequisites() {
    log "Checking prerequisites..."

    # Check kubectl
    if ! command -v kubectl >/dev/null 2>&1; then
        error "kubectl is required but not installed"
        exit 1
    fi

    # Check cluster connectivity
    if ! kubectl cluster-info >/dev/null 2>&1; then
        error "kubectl cannot connect to cluster"
        exit 1
    fi

    # Check git
    if ! command -v git >/dev/null 2>&1; then
        error "git is required but not installed"
        exit 1
    fi

    # Check if we're in a git repository
    if ! git rev-parse --git-dir >/dev/null 2>&1; then
        warn "Not in a git repository, using default repo URL"
    fi

    success "Prerequisites check passed"
}

setup_namespace() {
    log "Setting up namespace: $NAMESPACE"

    if ! kubectl get namespace "$NAMESPACE" >/dev/null 2>&1; then
        log "Creating namespace: $NAMESPACE"
        kubectl create namespace "$NAMESPACE"
    fi

    success "Namespace ready: $NAMESPACE"
}

setup_minio() {
    if [ "$SETUP_MINIO" = false ]; then
        log "Skipping MinIO setup"
        return
    fi

    log "Setting up MinIO for build caching..."

    # Apply MinIO resources
    kubectl apply -f "$SCRIPT_DIR/tekton/setup-minio.yaml"

    # Wait for MinIO to be ready
    log "Waiting for MinIO to be ready..."
    kubectl wait --for=condition=available deployment/minio --timeout=300s

    # Wait for bucket setup job to complete
    log "Waiting for bucket setup to complete..."
    kubectl wait --for=condition=complete job/minio-setup-bucket --timeout=120s || {
        warn "Bucket setup job didn't complete, but continuing..."
    }

    success "MinIO setup complete"
}

apply_tekton_resources() {
    if [ "$SETUP_TEKTON" = false ]; then
        log "Skipping Tekton resource setup"
        return
    fi

    log "Applying Tekton resources..."

    # Apply all Tekton resources
    for file in "$SCRIPT_DIR/tekton"/*.yaml; do
        if [[ "$(basename "$file")" != "setup-minio.yaml" && "$(basename "$file")" != "pipelinerun-local-dev.yaml" ]]; then
            log "Applying $(basename "$file")..."
            kubectl apply -f "$file"
        fi
    done

    success "Tekton resources applied"
}

run_build() {
    log "Starting StackRox build..."
    log "Registry: $REGISTRY"
    log "Tag: $TAG"
    log "Repository: $REPO_URL"
    log "Revision: $REVISION"

    # Create PipelineRun from template with substituted values
    cat "$SCRIPT_DIR/tekton/pipelinerun-local-dev.yaml" | \
    sed "s|value: \"https://github.com/stackrox/stackrox.git\"|value: \"$REPO_URL\"|g" | \
    sed "s|value: \"master\"|value: \"$REVISION\"|g" | \
    sed "s|value: \"localhost:5000\"|value: \"$REGISTRY\"|g" | \
    sed "s|value: \"latest\"|value: \"$TAG\"|g" | \
    sed "s|value: \"quay.io/stackrox-io/apollo-ci:stackrox-build-0.4.9\"|value: \"$BUILDER_IMAGE\"|g" | \
    sed "s|value: \"local-dev-cache\"|value: \"$CACHE_BUCKET\"|g" | \
    sed "s|value: \"minio.default.svc:9000\"|value: \"$MINIO_HOST\"|g" | \
    kubectl apply -f -

    # Get the PipelineRun name
    PIPELINERUN_NAME=$(kubectl get pipelinerun -n "$NAMESPACE" --sort-by=.metadata.creationTimestamp -o jsonpath='{.items[-1].metadata.name}')
    log "Started PipelineRun: $PIPELINERUN_NAME"

    if [ "$WAIT_FOR_COMPLETION" = true ]; then
        log "Waiting for pipeline to complete..."

        # Follow logs if verbose
        if [ "$VERBOSE" = true ]; then
            kubectl logs -f pipelinerun/"$PIPELINERUN_NAME" -n "$NAMESPACE" &
            LOG_PID=$!
        fi

        # Wait for completion
        kubectl wait --for=condition=Succeeded pipelinerun/"$PIPELINERUN_NAME" -n "$NAMESPACE" --timeout=1800s || {
            error "Pipeline failed or timed out"
            if [ "$VERBOSE" = true ] && [ -n "${LOG_PID:-}" ]; then
                kill $LOG_PID 2>/dev/null || true
            fi

            # Show failed task logs
            log "Showing logs from failed tasks:"
            kubectl describe pipelinerun/"$PIPELINERUN_NAME" -n "$NAMESPACE"
            exit 1
        }

        if [ "$VERBOSE" = true ] && [ -n "${LOG_PID:-}" ]; then
            kill $LOG_PID 2>/dev/null || true
        fi

        success "Pipeline completed successfully!"

        # Show final image references
        log "Built images:"
        GIT_SHA=$(kubectl get pipelinerun/"$PIPELINERUN_NAME" -n "$NAMESPACE" -o jsonpath='{.status.results[?(@.name=="git-commit")].value}' 2>/dev/null || echo "unknown")
        echo "  📦 $REGISTRY/stackrox/main:$GIT_SHA"
        echo "  📦 $REGISTRY/stackrox/main:$TAG"
    else
        log "Pipeline started: $PIPELINERUN_NAME"
        log "Monitor with: kubectl logs -f pipelinerun/$PIPELINERUN_NAME -n $NAMESPACE"
    fi
}

setup_helm_integration() {
    log "Setting up Helm integration..."

    # Set environment variables for local-dev flavor
    export ROX_IMAGE_FLAVOR=local-dev
    export ROX_LOCAL_REGISTRY="$REGISTRY"
    export ROX_LOCAL_TAG="$TAG"

    echo ""
    success "Helm integration configured!"
    echo "Environment variables set:"
    echo "  ROX_IMAGE_FLAVOR=local-dev"
    echo "  ROX_LOCAL_REGISTRY=$REGISTRY"
    echo "  ROX_LOCAL_TAG=$TAG"
    echo ""
    echo "To deploy StackRox with your custom images:"
    echo "  export ROX_IMAGE_FLAVOR=local-dev"
    echo "  export ROX_LOCAL_REGISTRY=$REGISTRY"
    echo "  export ROX_LOCAL_TAG=$TAG"
    echo "  cd ./installer"
    echo "  go build -o bin/installer ./installer"
    echo "  ./bin/installer apply central"
}

deploy_stackrox() {
    if [ "$DEPLOY_STACKROX" = false ]; then
        return
    fi

    log "Deploying StackRox..."

    # Set up environment for local-dev flavor
    export ROX_IMAGE_FLAVOR=local-dev
    export ROX_LOCAL_REGISTRY="$REGISTRY"
    export ROX_LOCAL_TAG="$TAG"

    # Check if installer exists
    if [ ! -f "$PROJECT_ROOT/installer/bin/installer" ]; then
        log "Building installer..."
        cd "$PROJECT_ROOT/installer"
        go build -o bin/installer ./installer
        cd "$PROJECT_ROOT"
    fi

    # Deploy central
    log "Deploying Central..."
    "$PROJECT_ROOT/installer/bin/installer" apply central

    # Wait for central to be ready
    log "Waiting for Central to be ready..."
    kubectl -n stackrox wait --for=condition=available deployment/central --timeout=600s

    success "StackRox deployment complete!"
    echo ""
    echo "Access StackRox Central:"
    echo "  kubectl -n stackrox port-forward svc/central 8443:443"
    echo "  Open: https://localhost:8443"
    echo "  Username: admin"
    echo "  Password: letmein"
}

parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            -r|--registry)
                REGISTRY="$2"
                shift 2
                ;;
            -t|--tag)
                TAG="$2"
                shift 2
                ;;
            -n|--namespace)
                NAMESPACE="$2"
                shift 2
                ;;
            --repo-url)
                REPO_URL="$2"
                shift 2
                ;;
            --revision)
                REVISION="$2"
                shift 2
                ;;
            --builder-image)
                BUILDER_IMAGE="$2"
                shift 2
                ;;
            --cache-bucket)
                CACHE_BUCKET="$2"
                shift 2
                ;;
            --minio-host)
                MINIO_HOST="$2"
                shift 2
                ;;
            --no-setup-minio)
                SETUP_MINIO=false
                shift
                ;;
            --no-setup-tekton)
                SETUP_TEKTON=false
                shift
                ;;
            --no-wait)
                WAIT_FOR_COMPLETION=false
                shift
                ;;
            --deploy)
                DEPLOY_STACKROX=true
                shift
                ;;
            -v|--verbose)
                VERBOSE=true
                shift
                ;;
            -h|--help)
                usage
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                usage
                exit 1
                ;;
        esac
    done
}

main() {
    echo "🚀 Fast StackRox Tekton Build Tool"
    echo ""

    parse_args "$@"
    check_prerequisites
    setup_namespace
    setup_minio
    apply_tekton_resources
    run_build
    setup_helm_integration
    deploy_stackrox

    echo ""
    success "🎉 Build complete! Your local StackRox images are ready."
    echo ""
    echo "Next steps:"
    echo "  1. Verify images: docker images | grep $REGISTRY"
    echo "  2. Deploy: export ROX_IMAGE_FLAVOR=local-dev && ./installer/bin/installer apply central"
    echo "  3. Access: kubectl -n stackrox port-forward svc/central 8443:443"
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi