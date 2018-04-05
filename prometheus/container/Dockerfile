FROM prom/prometheus:v2.2.1
MAINTAINER StackRox <support@stackrox.com>

COPY entrypoint.sh /entrypoint.sh
COPY prometheus.yml.template /etc/prometheus/prometheus.yml.template

ENTRYPOINT ["/entrypoint.sh"]