FROM alpine:3.8

ENV INFLUXDB_VERSION=1.6.2

RUN wget https://dl.influxdata.com/influxdb/releases/influxdb-${INFLUXDB_VERSION}-static_linux_amd64.tar.gz && \
    tar xvfz influxdb-${INFLUXDB_VERSION}-static_linux_amd64.tar.gz && mv /influxdb-${INFLUXDB_VERSION}-1/influxd /influxd && \
    mv /influxdb-${INFLUXDB_VERSION}-1/influx /influx


ENV KAPACITOR_VERSION=1.5.1

RUN wget https://dl.influxdata.com/kapacitor/releases/kapacitor-${KAPACITOR_VERSION}-static_linux_amd64.tar.gz && \
    tar xvfz kapacitor-${KAPACITOR_VERSION}-static_linux_amd64.tar.gz && mv kapacitor-${KAPACITOR_VERSION}-1/kapacitord /kapacitord


ENV CHRONOGRAF_VERSION=1.6.2

RUN wget https://dl.influxdata.com/chronograf/releases/chronograf-${CHRONOGRAF_VERSION}-static_linux_amd64.tar.gz && \
    tar xvfz chronograf-${CHRONOGRAF_VERSION}-static_linux_amd64.tar.gz && mv chronograf-${CHRONOGRAF_VERSION}-1/chronograf /chronograf

COPY ./influx-post-start.sh /influx-post-start.sh

COPY ./kapacitor.sh /kapacitor.sh
