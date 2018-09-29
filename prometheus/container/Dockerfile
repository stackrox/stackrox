FROM prom/prometheus:v2.4.2
LABEL "com.vendor"="StackRox (support@stackrox.com)"

COPY entrypoint.sh /entrypoint.sh
COPY prometheus.yml.template /etc/prometheus/prometheus.yml.template

ENTRYPOINT ["/entrypoint.sh"]
