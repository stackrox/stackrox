# Docker Buildx bake file for consolidated component builds
# Builds: main, scanner, operator, roxctl
#
# Usage:
#   docker buildx bake -f .github/docker/components-bake.hcl --print all
#   docker buildx bake -f .github/docker/components-bake.hcl --push all

variable "TAG" {
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

variable "MAIN_REGISTRY" {
  default = "quay.io/stackrox-io"
}

variable "SCANNER_REGISTRY" {
  default = "quay.io/stackrox-io"
}

variable "OPERATOR_REGISTRY" {
  default = "quay.io/stackrox-io"
}

variable "ROXCTL_REGISTRY" {
  default = "quay.io/stackrox-io"
}

variable "LABEL_VERSION" {
  default = ""
}

variable "LABEL_RELEASE" {
  default = ""
}

variable "QUAY_TAG_EXPIRATION" {
  default = "14d"
}

# Group to build all components
group "all" {
  targets = ["main", "scanner", "operator", "roxctl"]
}

# Main image - central, sensor, etc.
target "main" {
  context = "image/rhel"
  dockerfile = "Dockerfile"
  platforms = split(",", PLATFORMS)
  tags = [
    "${MAIN_REGISTRY}/main:${TAG}"
  ]
  args = {
    DEBUG_BUILD = DEBUG_BUILD
    ROX_PRODUCT_BRANDING = ROX_PRODUCT_BRANDING
    ROX_IMAGE_FLAVOR = ROX_IMAGE_FLAVOR
    LABEL_VERSION = notequal("", LABEL_VERSION) ? LABEL_VERSION : TAG
    LABEL_RELEASE = notequal("", LABEL_RELEASE) ? LABEL_RELEASE : TAG
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
    "${SCANNER_REGISTRY}/scanner:${TAG}"
  ]
  args = {
    DEBUG_BUILD = DEBUG_BUILD
    LABEL_VERSION = notequal("", LABEL_VERSION) ? LABEL_VERSION : TAG
    LABEL_RELEASE = notequal("", LABEL_RELEASE) ? LABEL_RELEASE : TAG
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
    "${OPERATOR_REGISTRY}/stackrox-operator:${TAG}"
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
    "${ROXCTL_REGISTRY}/roxctl:${TAG}"
  ]
  cache-from = ["type=gha,scope=roxctl"]
  cache-to = ["type=gha,mode=max,scope=roxctl"]
}
