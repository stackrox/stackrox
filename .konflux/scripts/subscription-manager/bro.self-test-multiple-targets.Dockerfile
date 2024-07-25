# This verifies that subscription-manager-bro.sh can prepare multiple distinct target stages from one installer stage.

# We use the same base as target for both stages, but these could as well be different.
FROM registry.access.redhat.com/ubi9/ubi-minimal:latest AS target-base

# Sanity-check we don't have any of the target packages already.
RUN ! rpm -q bpftool && \
    ! rpm -q snappy && \
    ! microdnf -y install bpftool && \
    ! microdnf -y install snappy


FROM registry.access.redhat.com/ubi9/ubi:latest AS installer

COPY --from=target-base / /mnt/stage1
COPY --from=target-base / /mnt/stage2

COPY ./subscription-manager-bro.sh /tmp/.konflux/
COPY ./activation-key /activation-key/

# Register both target paths at once.
RUN /tmp/.konflux/subscription-manager-bro.sh register /mnt/stage1 /mnt/stage2

# Install a package to the first target,
RUN dnf -y --installroot=/mnt/stage1 install bpftool && \
    rpm --root=/mnt/stage1 -q bpftool && ! rpm --root=/mnt/stage1 -q snappy && \
    dnf -y --installroot=/mnt/stage1 remove bpftool && dnf -y --installroot=/mnt/stage1 autoremove

# and another to the second one.
RUN dnf -y --installroot=/mnt/stage2 install snappy && \
    rpm --root=/mnt/stage2 -q snappy && ! rpm --root=/mnt/stage2 -q bpftool && \
    dnf -y --installroot=/mnt/stage2 remove snappy && dnf -y --installroot=/mnt/stage2 autoremove

# The cleanup triggered with a single command happens for everything previously registered.
RUN /tmp/.konflux/subscription-manager-bro.sh cleanup


FROM installer AS assert-no-significant-diff-after-cleanup

RUN dnf -y install diffutils less
COPY ./ /tmp/.konflux

COPY --from=target-base / /mnt/expected/

COPY --from=installer /mnt/stage1 /mnt/actual1/
COPY --from=installer /mnt/stage2 /mnt/actual2/

RUN /tmp/.konflux/subscription-manager-bro.sh diff /mnt/expected /mnt/actual1
RUN /tmp/.konflux/subscription-manager-bro.sh diff /mnt/expected /mnt/actual2
