FROM registry.access.redhat.com/ubi8-micro:8.7

ARG ROX_IMAGE_FLAVOR
ENV ROX_IMAGE_FLAVOR=${ROX_IMAGE_FLAVOR}

# The following are numeric uid and gid of `nobody` user in UBI.
# We can't use symbolic names because otherwise k8s will fail to start the pod with an error like this:
# Error: container has runAsNonRoot and image has non-numeric user (nobody), cannot verify user is non-root (pod: "stackrox-operator-controller-manager-75bc744454-bkbjr_stackrox-operator-system(49874aae-2695-4d3a-afd3-8723914d2af5)", container: manager)
USER 65534:65534

ENTRYPOINT ["/usr/local/bin/stackrox-operator"]

COPY bin/manager /usr/local/bin/stackrox-operator
