# Pushed to: "quay.io/rhacs-eng/qa:dele-scan-httpd".
FROM docker.io/library/httpd:2.4.62

# httpd by default will bind to port 80 which will cause permissions issues
# in restricted envs.  Since we do not care about httpd functionality for the
# associated tests, we instead sleep forever.  As an alternative the httpd
# config could be updated to bind to a differnet port, however sleep
# required less effort to implement.

# src: tests/testdata/delegatedscanning/image-build/sleepforever.sh
COPY --chmod=755 ./sleepforever.sh /sleepforever.sh

ENTRYPOINT [ "/sleepforever.sh" ]
