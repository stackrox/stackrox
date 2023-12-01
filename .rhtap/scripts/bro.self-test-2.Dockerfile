# This demonstrates usage of subscription-manager-bro.sh and verifies important assumptions.

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS target_base

# Sanity-check we don't have the target package already.
RUN ! psql --version && \
    ! rpm -q postgresql && \
    ! microdnf install postgresql


FROM registry.access.redhat.com/ubi8/ubi:latest AS installer

COPY --from=target_base / /mnt
COPY ./.rhtap /tmp/.rhtap

# Sanity-check it's entitled.
RUN ! dnf -y --installroot=/mnt module enable -y postgresql:15 && \
    ! dnf -y --installroot=/mnt install postgresql

# Here's how to use `register` and `cleanup` subcommands.
RUN /tmp/.rhtap/scripts/subscription-manager-bro.sh register && \
    dnf -y --installroot=/mnt module enable -y postgresql:15 && \
    dnf -y --installroot=/mnt install postgresql && \
    /tmp/.rhtap/scripts/subscription-manager-bro.sh cleanup


FROM scratch AS target

# This makes this `target` stage as desired: target_base + postgresql
COPY --from=installer /mnt /

# The command must be found.
RUN psql --version

# rpmdb must contain an entry for the package.
RUN rpm -q postgresql

# There must be no way to further install entitled packages.
RUN microdnf repolist && ! microdnf install snappy
