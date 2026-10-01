# Docker Buildx bake file for container images
# Builds: central-db, scanner-v4-db
#
# Usage:
#   docker buildx bake -f .github/docker/images.hcl --print databases
#   docker buildx bake -f .github/docker/images.hcl --push databases
#
# Final cache verification

variable "TAG" {
  default = "latest"
}

variable "PLATFORMS" {
  default = "linux/amd64,linux/arm64"
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
