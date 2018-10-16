FROM alpine:3.8

ENV INFLUXDB_VERSION=1.6.2

RUN wget https://dl.influxdata.com/influxdb/releases/influxdb-${INFLUXDB_VERSION}-static_linux_amd64.tar.gz && \
    tar xvfz influxdb-${INFLUXDB_VERSION}-static_linux_amd64.tar.gz && mv /influxdb-${INFLUXDB_VERSION}-1/influxd /influxd && \
    mv /influxdb-${INFLUXDB_VERSION}-1/influx /influx

FROM grafana/grafana:5.3.0

COPY --from=0 /influx /influx
COPY --from=0 /influxd /influxd
COPY ./influx-post-start.sh /influx-post-start.sh

USER root
