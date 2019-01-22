FROM alpine:3.8

ENV INFLUXDB_VERSION=1.6.2

RUN wget https://dl.influxdata.com/influxdb/releases/influxdb-${INFLUXDB_VERSION}-static_linux_amd64.tar.gz && \
    tar xvfz influxdb-${INFLUXDB_VERSION}-static_linux_amd64.tar.gz && mv /influxdb-${INFLUXDB_VERSION}-1/influxd /influxd && \
    mv /influxdb-${INFLUXDB_VERSION}-1/influx /influx

RUN wget https://github.com/connorgorman/telegraf/releases/download/1.8.3%2B179-slim/telegraf -O /telegraf && chmod +x /telegraf

FROM grafana/grafana:5.3.0

COPY --from=0 /influx /influx
COPY --from=0 /influxd /influxd
COPY --from=0 /telegraf /telegraf

COPY ./influxdb/influx-post-start.sh /influx-post-start.sh


COPY ./grafana/dashboard-config.yaml /etc/grafana/provisioning/dashboards/dashboard-config.yaml
COPY ./grafana/dashboards/* /etc/grafana/dashboards/
COPY ./grafana/datasources/* /etc/grafana/provisioning/datasources/
COPY ./grafana/entrypoint.sh /grafana-entrypoint.sh

USER root
