FROM registry.access.redhat.com/ubi8/nodejs-18:latest AS ui-builder

# Switch to root because ubi8/nodejs image runs as non-root user by default which does not let install RPMs.
USER 0:0

WORKDIR /go/src/github.com/stackrox/rox/app

COPY . .

# This sets branding during UI build time. This is to make sure UI is branded as commercial RHACS (not StackRox).
# ROX_PRODUCT_BRANDING is also set in the resulting image so that Central Go code knows its RHACS.
ENV ROX_PRODUCT_BRANDING="RHACS_BRANDING"

# This installs yarn from Cachi2.
RUN cd image/rhel/rhtap-bootstrap-yarn && npm install --global

# UI build is not hermetic because Cachi2 does not support pulling packages according to yarn.lock yet.
# TODO(ROX-20723): make UI builds hermetic when Cachi2 supports that.
#RUN dnf install -y make patch
RUN make -C ui build
