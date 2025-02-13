FROM registry.access.redhat.com/ubi8/nodejs-20:latest AS ui-builder

WORKDIR /go/src/github.com/stackrox/rox/app

COPY --chown=default . .

# This sets branding during UI build time. This is to make sure UI is branded as commercial RHACS (not StackRox).
# ROX_PRODUCT_BRANDING is also set in the resulting image so that Central Go code knows its RHACS.
ENV ROX_PRODUCT_BRANDING="RHACS_BRANDING"

# Default execution of the `npm ci` command causes postinstall scripts to run and spawn a new child process
# for each script. When building in konflux for s390x and ppc64le architectures, spawing
# these child processes causes excessive memory usage and ENOMEM errors, resulting
# in build failures. Currently the only postinstall scripts that run for the UI dependencies are:
#   `core-js` prints a banner with links for donations
#   `cypress` downloads the Cypress binary from the internet
# In the case of building the `rhacs-main-container`, all of these install scripts can be safely ignored.
ENV UI_PKG_INSTALL_EXTRA_ARGS="--ignore-scripts"

RUN make -C ui build
