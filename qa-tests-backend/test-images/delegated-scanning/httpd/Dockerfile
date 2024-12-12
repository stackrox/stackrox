# This base image was chosen randomly due to smallish size, is self running, and contains
# at least one vulnerability. For some of the delegated scanning tests it's important the
# image under test hasn't been scanned yet to accurately validate behaviors, this is
# one of multiple 'unique' images. This image should ONLY be used by the delegated
# scanning tests.
#
# After build the resulting image was pushed to: "quay.io/rhacs-eng/qa:dele-scan-httpd".

FROM docker.io/library/httpd:2.4.62

# Change the port to a non-prvileged port so that the image will run in restricted envs.
RUN sed -i 's/Listen 80/Listen 8080/g' /usr/local/apache2/conf/httpd.conf

CMD ["httpd-foreground", "-C", "PidFile /tmp/httpd.pid"]
