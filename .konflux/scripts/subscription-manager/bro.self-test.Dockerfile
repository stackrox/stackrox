# This tests ability of subscription-manager-bro.sh to install entitled packages cleanly.

ARG TEST_RHEL_PACKAGE=snappy

ARG TARGET_BASE=registry.access.redhat.com/ubi8/ubi-micro:latest
ARG INSTALLER_MAJOR_VERSION=8

FROM $TARGET_BASE AS target-base
# This stage must stay empty so we find 100% original container aliased as target-base.


# The installer must be ubi (not minimal) and must be 8.9 or later since the earlier versions complain:
#  subscription-manager is disabled when running inside a container. Please refer to your host system for subscription management.
FROM registry.access.redhat.com/ubi${INSTALLER_MAJOR_VERSION}/ubi:latest AS installer


FROM installer AS test-no-entitlement
COPY --from=target-base / /mnt
ARG TEST_RHEL_PACKAGE
RUN ! dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE"


FROM installer AS test-yes-entitlement

COPY --from=target-base / /mnt
COPY ./ /tmp/.konflux
COPY ./activation-key /activation-key/

ARG TEST_RHEL_PACKAGE
RUN /tmp/.konflux/subscription-manager-bro.sh register /mnt && \
    dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE" && \
    dnf -y --installroot=/mnt remove "$TEST_RHEL_PACKAGE" && \
    /tmp/.konflux/subscription-manager-bro.sh cleanup


FROM installer AS assert-no-significant-diff-after-cleanup

RUN dnf -y install diffutils less
COPY ./ /tmp/.konflux

COPY --from=target-base / /mnt/expected/
COPY --from=test-yes-entitlement /mnt /mnt/actual/

RUN /tmp/.konflux/subscription-manager-bro.sh diff /mnt/expected /mnt/actual
