# Docker Buildx bake file for database images
# Builds: central-db, scanner-v4-db
#
# Usage:
#   docker buildx bake -f .github/docker/db-bake.hcl --print all
#   docker buildx bake -f .github/docker/db-bake.hcl --push all

variable "TAG" {
  default = "latest"
}

variable "PLATFORMS" {
  default = "linux/amd64,linux/arm64"
}

variable "REGISTRY" {
  default = "quay.io/stackrox-io"
}

variable "LABEL_VERSION" {
  default = ""
}

variable "LABEL_RELEASE" {
  default = ""
}

# Group to build all DB images
group "all" {
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
    LABEL_VERSION = notequal("", LABEL_VERSION) ? LABEL_VERSION : TAG
    LABEL_RELEASE = notequal("", LABEL_RELEASE) ? LABEL_RELEASE : TAG
  }
  cache-from = ["type=gha,scope=scanner-v4-db"]
  cache-to = ["type=gha,mode=max,scope=scanner-v4-db"]
}
