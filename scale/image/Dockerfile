FROM google/cloud-sdk:289.0.0

# Profiler
RUN apt-get update && apt-get -y install wget curl python2.7-minimal libpython-stdlib

COPY ./image/bin/profiler /profiler
COPY ./profiler/pprof.sh /pprof.sh
COPY ./profiler/entrypoint.sh /profiler-entrypoint.sh

COPY ./image/bin/mocksensor /mocksensor
COPY ./image/bin/mockcollector /mockcollector
COPY ./image/bin/chaos /chaos

COPY ./image/bin/flightreplay /flightreplay
COPY ./image/docker-entrypoint.sh /docker-entrypoint.sh

COPY ./image/fixtures/scrape.json /files/scrape.json
