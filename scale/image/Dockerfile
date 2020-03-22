FROM ubuntu:16.04

# Profiler
RUN apt-get update && apt-get -y install wget curl python2.7-minimal libpython-stdlib

RUN curl -O https://dl.google.com/dl/cloudsdk/channels/rapid/downloads/google-cloud-sdk-258.0.0-linux-x86_64.tar.gz && \
    tar zxvf google-cloud-sdk-258.0.0-linux-x86_64.tar.gz google-cloud-sdk && \
    ./google-cloud-sdk/install.sh

COPY ./image/bin/profiler /profiler
COPY ./profiler/pprof.sh /pprof.sh
COPY ./profiler/entrypoint.sh /profiler-entrypoint.sh

COPY ./image/bin/mocksensor /mocksensor
COPY ./image/bin/mockcollector /mockcollector

COPY ./image/bin/flightreplay /flightreplay
COPY ./image/docker-entrypoint.sh /docker-entrypoint.sh

COPY ./image/fixtures/scrape.json /files/scrape.json
