FROM registry.access.redhat.com/ubi9/nodejs-16:latest AS builder

USER 0:0

WORKDIR /workspace
COPY content content
COPY tools tools
COPY Makefile ./

RUN npm install -g npm@8.10.0 && npm install -g yarn@1.22.18
RUN make

FROM registry.access.redhat.com/ubi9/nginx-120:latest

RUN echo "$(id -u):$(id -g)" > /tmp/container-uid

# Update RPMs to hopefully aid with vulns that aren't yet patched in the base image. Also remove package management
# utils because our own policies don't recommend having them in the image. For that we need to be root.
USER 0:0
RUN dnf upgrade -y && \
    dnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

# Switch back to the normal user of nginx container and check that we really switch to the original one.
USER 1001:0
RUN echo "If the following command fails, update USER statement to match UID:GID of the user of nginx base container." && \
    [[ "$(id -u):$(id -g)" == "$(cat /tmp/container-uid)" ]] && \
    rm -f /tmp/container-uid

# Delete unnecessary things from the content directory.
RUN find . -mindepth 1 -delete

COPY --from=builder /workspace/build/ ./

EXPOSE 8080

CMD nginx -g "daemon off;"
