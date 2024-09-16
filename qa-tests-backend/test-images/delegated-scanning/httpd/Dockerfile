# After build the resulting image was pushed to: "quay.io/rhacs-eng/qa:dele-scan-httpd".
#
# The resulting image should ONLY be used by the delegated scanning tests.

FROM docker.io/library/httpd:2.4.62

RUN sed -i 's/Listen 80/Listen 8080/g' /usr/local/apache2/conf/httpd.conf

CMD ["httpd-foreground", "-C", "PidFile /tmp/httpd.pid"]
