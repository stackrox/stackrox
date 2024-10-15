# This base image was chosen randomly due to smallish size, is self running, and contains
# at least one vulnerability. For some of the delegated scanning tests it's important the
# image under test hasn't been scanned yet to accurately validate behaviors, this is
# one of multiple 'unique' images. This image should ONLY be used by the delegated
# scanning tests.
#
# After build the resulting image was pushed to: "quay.io/rhacs-eng/qa:dele-scan-memcached".

FROM docker.io/library/memcached:1.6.31
