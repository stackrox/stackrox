# This tests ability of subscription-manager-bro.sh to install entitled packages cleanly.

ARG TEST_RHEL_PACKAGE=snappy

FROM registry.access.redhat.com/ubi8/ubi-micro:latest AS ubi-micro
# Must stay empty so we find 100% original container aliased as ubi-normal.
FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS ubi-minimal
# Must stay empty.
FROM registry.access.redhat.com/ubi8/ubi:latest AS ubi-normal
# Must stay empty.
FROM registry.redhat.io/rhel8/pause AS rhel
# Must stay empty.


# The installer must be ubi (not minimal) and must be 8.9 or later since the earlier versions complain:
#  subscription-manager is disabled when running inside a container. Please refer to your host system for subscription management.
FROM registry.access.redhat.com/ubi8/ubi:latest AS installer


FROM installer AS test-no-entitlement-micro
COPY --from=ubi-micro / /mnt
ARG TEST_RHEL_PACKAGE
RUN ! dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE"


FROM installer AS test-no-entitlement-minimal
COPY --from=ubi-minimal / /mnt
ARG TEST_RHEL_PACKAGE
RUN ! dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE"


FROM installer AS test-no-entitlement-normal
COPY --from=ubi-normal / /mnt
ARG TEST_RHEL_PACKAGE
RUN ! dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE"


FROM installer AS test-no-entitlement-rhel
COPY --from=rhel / /mnt
ARG TEST_RHEL_PACKAGE
RUN ! dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE"


FROM installer AS test-yes-entitlement-micro

COPY --from=ubi-micro / /mnt
COPY ./.rhtap /tmp/.rhtap

ARG TEST_RHEL_PACKAGE
RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh register && \
    dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE" && \
    dnf -y --installroot=/mnt remove "$TEST_RHEL_PACKAGE" && \
    /tmp/.rhtap/scripts/subscription-manager-bro.sh cleanup


FROM installer AS test-yes-entitlement-minimal

COPY --from=ubi-minimal / /mnt
COPY ./.rhtap /tmp/.rhtap

ARG TEST_RHEL_PACKAGE
RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh register && \
    dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE" && \
    dnf -y --installroot=/mnt remove "$TEST_RHEL_PACKAGE" && \
    /tmp/.rhtap/scripts/subscription-manager-bro.sh cleanup


FROM installer AS test-yes-entitlement-normal

COPY --from=ubi-normal / /mnt
COPY ./.rhtap /tmp/.rhtap

ARG TEST_RHEL_PACKAGE
RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh register && \
    dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE" && \
    dnf -y --installroot=/mnt remove "$TEST_RHEL_PACKAGE" && \
    /tmp/.rhtap/scripts/subscription-manager-bro.sh cleanup


FROM installer AS test-yes-entitlement-rhel

COPY --from=rhel / /mnt
COPY ./.rhtap /tmp/.rhtap

ARG TEST_RHEL_PACKAGE
RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh register && \
    dnf -y --installroot=/mnt install "$TEST_RHEL_PACKAGE" && \
    dnf -y --installroot=/mnt remove "$TEST_RHEL_PACKAGE" && \
    /tmp/.rhtap/scripts/subscription-manager-bro.sh cleanup


FROM ubi-normal AS assert-no-significant-diff-after-cleanup

RUN dnf -y install diffutils
COPY ./.rhtap /tmp/.rhtap

COPY --from=ubi-micro / /mnt/micro-expected/
COPY --from=test-yes-entitlement-micro /mnt /mnt/micro-actual/

RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh diff /mnt/micro-expected /mnt/micro-actual

COPY --from=ubi-minimal / /mnt/minimal-expected/
COPY --from=test-yes-entitlement-minimal /mnt /mnt/minimal-actual/

RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh diff /mnt/minimal-expected /mnt/minimal-actual

COPY --from=ubi-normal / /mnt/normal-expected/
COPY --from=test-yes-entitlement-normal /mnt /mnt/normal-actual/

RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh diff /mnt/normal-expected /mnt/normal-actual

COPY --from=rhel / /mnt/rhel-expected/
COPY --from=test-yes-entitlement-rhel /mnt /mnt/rhel-actual/

RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh diff /mnt/rhel-expected /mnt/rhel-actual
