FROM google/cloud-sdk:486.0.0

# Profiler
RUN apt-get update && apt-get -y install wget curl

COPY ./image/bin/profiler /profiler
COPY ./profiler/pprof.sh /pprof.sh
COPY ./profiler/entrypoint.sh /profiler-entrypoint.sh

COPY ./image/bin/chaos /chaos

COPY ./image/fixtures/scrape.json /files/scrape.json
COPY ./image/fixtures/results.json /files/results.json
