FROM registry.access.redhat.com/ubi8/nodejs-16 AS builder

USER 0:0

WORKDIR /workspace
COPY content content
COPY tools tools
COPY Makefile ./

RUN npm install -g npm@8.1.4 && npm install -g yarn@1.22.17
RUN make

FROM registry.access.redhat.com/ubi8/nginx-120:latest

# Delete unnecessary things from the content directory.
RUN find . -mindepth 1 -delete

# Update RPMs to hopefully aid with vulns that aren't yet patched in the base image. Also remove package management
# utils because our own policies don't recommend having them in the image.
RUN dnf upgrade -y && \
    dnf clean all && \
    rpm --verbose -e --nodeps $(rpm -qa curl '*rpm*' '*dnf*' '*libsolv*' '*hawkey*' 'yum*') && \
    rm -rf /var/cache/dnf /var/cache/yum

COPY --from=builder /workspace/build/ ./

EXPOSE 8080

CMD nginx -g "daemon off;"
