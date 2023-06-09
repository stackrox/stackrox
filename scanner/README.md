# StackRox Scanner

Static Image, Node, and Orchestrator Scanner.

# Dev

Scanner requires go1.20+

# Updating Build Image

For consistency across CI and development, we try to use a common base image
when building Scanner and ScannerDB.

BUILD_IMAGE_VERSION reflects the image used to build Scanner and ScannerDB;
however, it is not the only location where the build image is defined.

To update the build image, be sure to update the following files:

* .github/workflows/build-scanner.yaml
* scanner/BUILD_IMAGE_VERSION
