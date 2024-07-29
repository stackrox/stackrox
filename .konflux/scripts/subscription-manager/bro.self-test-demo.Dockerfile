# This demonstrates the usage of subscription-manager-bro.sh and verifies important assumptions.

FROM registry.access.redhat.com/ubi8/ubi-minimal:latest AS target-base

# Sanity-check we don't have the target package already.
RUN ! psql --version && \
    ! rpm -q postgresql && \
    ! microdnf -y module enable postgresql:15 && \
    ! microdnf -y install postgresql


FROM registry.access.redhat.com/ubi8/ubi:latest AS installer

# Put the target image into installer's /mnt. That's where it will be manipulated by the installer.
COPY --from=target-base / /mnt

# Copy our helper script and the activation key to the installer.
COPY ./subscription-manager-bro.sh /tmp/.konflux/
COPY ./activation-key /activation-key/

# Sanity-check the installer is not entitled (yet).
RUN ! dnf -y --installroot=/mnt module enable postgresql:15 && \
    ! dnf -y --installroot=/mnt install postgresql

# Here's how to use `register` and `cleanup` subcommands.
RUN /tmp/.konflux/subscription-manager-bro.sh register /mnt && \
    dnf -y --installroot=/mnt module enable postgresql:15 && \
    dnf -y --installroot=/mnt install postgresql && \
    /tmp/.konflux/subscription-manager-bro.sh cleanup


FROM scratch AS target

# This makes this `target` stage as desired: target_base + postgresql
COPY --from=installer /mnt /

# The command must be found.
RUN psql --version

# rpmdb must contain an entry for the package.
RUN rpm -q postgresql

# There must be no way to further install entitled packages.
RUN microdnf repolist && ! microdnf -y install snappy
