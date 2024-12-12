# Lineage Test Images
The images produced by these Dockerfiles have common top and bottom layers, the middle layers differ.

This scenario has caused past scanning issues, refer to ROX-26604 for more details.

## Building

```sh
cd qa-tests-backend/test-images/lineage

docker build -t quay.io/rhacs-eng/qa:lineage-jdk-17.0.11 -f Dockerfile.jdk17.0.11 .

docker build -t quay.io/rhacs-eng/qa:lineage-jdk-17.0.13 -f Dockerfile.jdk17.0.13 .
```