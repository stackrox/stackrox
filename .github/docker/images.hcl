# Docker Buildx bake file for all container images
# Builds: databases (central-db, scanner-v4-db) and components (main, scanner, operator, roxctl)
#
# Usage:
#   docker buildx bake -f .github/docker/images.hcl --print databases
#   docker buildx bake -f .github/docker/images.hcl --print components
#   docker buildx bake -f .github/docker/images.hcl --push databases
#   docker buildx bake -f .github/docker/images.hcl --push components

variable "TAG" {
  default = "latest"
}

variable "TAG_OPERATOR" {
  default = "latest"
}

variable "PLATFORMS" {
  default = "linux/amd64,linux/arm64"
}

variable "ROX_PRODUCT_BRANDING" {
  default = "STACKROX_BRANDING"
}

variable "ROX_IMAGE_FLAVOR" {
  default = "development_build"
}

variable "DEBUG_BUILD" {
  default = "no"
}

variable "REGISTRY" {
  default = "quay.io/stackrox-io"
}

variable "QUAY_TAG_EXPIRATION" {
  default = "13w"
}

# Group to build database images
group "databases" {
  targets = ["central-db", "scanner-v4-db"]
}

# Group to build component images
group "components" {
  targets = ["main", "scanner", "operator", "roxctl"]
}

# Central database (PostgreSQL 15)
target "central-db" {
  context = "image/postgres"
  dockerfile = "Dockerfile"
  platforms = split(",", PLATFORMS)
  tags = [
    "${REGISTRY}/central-db:${TAG}"
  ]
  args = {
    MAIN_IMAGE_TAG = TAG
    QUAY_TAG_EXPIRATION = QUAY_TAG_EXPIRATION
  }
  cache-from = ["type=gha,scope=central-db"]
  cache-to = ["type=gha,mode=max,scope=central-db"]
}

# Scanner V4 database (PostgreSQL 15)
target "scanner-v4-db" {
  context = "scanner/image/db"
  dockerfile = "Dockerfile"
  platforms = split(",", PLATFORMS)
  tags = [
    "${REGISTRY}/scanner-v4-db:${TAG}"
  ]
  args = {
    LABEL_VERSION = TAG
    LABEL_RELEASE = TAG
    QUAY_TAG_EXPIRATION = QUAY_TAG_EXPIRATION
  }
  cache-from = ["type=gha,scope=scanner-v4-db"]
  cache-to = ["type=gha,mode=max,scope=scanner-v4-db"]
}

# Main image - central, sensor, etc.
target "main" {
  context = "image/rhel"
  dockerfile = "Dockerfile"
  platforms = split(",", PLATFORMS)
  tags = [
    "${REGISTRY}/main:${TAG}"
  ]
  args = {
    DEBUG_BUILD = DEBUG_BUILD
    ROX_PRODUCT_BRANDING = ROX_PRODUCT_BRANDING
    ROX_IMAGE_FLAVOR = ROX_IMAGE_FLAVOR
    LABEL_VERSION = TAG
    LABEL_RELEASE = TAG
    QUAY_TAG_EXPIRATION = QUAY_TAG_EXPIRATION
  }
  cache-from = ["type=gha,scope=main-${ROX_PRODUCT_BRANDING}"]
  cache-to = ["type=gha,mode=max,scope=main-${ROX_PRODUCT_BRANDING}"]
}

# Scanner image
target "scanner" {
  context = "scanner/image/scanner"
  dockerfile = "Dockerfile"
  platforms = split(",", PLATFORMS)
  tags = [
    "${REGISTRY}/scanner:${TAG}"
  ]
  args = {
    DEBUG_BUILD = DEBUG_BUILD
    LABEL_VERSION = TAG
    LABEL_RELEASE = TAG
    QUAY_TAG_EXPIRATION = QUAY_TAG_EXPIRATION
  }
  cache-from = ["type=gha,scope=scanner"]
  cache-to = ["type=gha,mode=max,scope=scanner"]
}

# Operator image
target "operator" {
  context = "operator"
  dockerfile = "prebuilt.Dockerfile"
  platforms = split(",", PLATFORMS)
  tags = [
    "${REGISTRY}/stackrox-operator:${TAG_OPERATOR}"
  ]
  args = {
    ROX_IMAGE_FLAVOR = ROX_IMAGE_FLAVOR
  }
  cache-from = ["type=gha,scope=operator-${ROX_PRODUCT_BRANDING}"]
  cache-to = ["type=gha,mode=max,scope=operator-${ROX_PRODUCT_BRANDING}"]
}

# Roxctl CLI image
target "roxctl" {
  context = "image/roxctl"
  dockerfile = "Dockerfile"
  platforms = split(",", PLATFORMS)
  tags = [
    "${REGISTRY}/roxctl:${TAG}"
  ]
  cache-from = ["type=gha,scope=roxctl"]
  cache-to = ["type=gha,mode=max,scope=roxctl"]
}
