FROM alpine:3.8

COPY ./image/bin/mocksensor /mocksensor
COPY ./image/bin/mockcollector /mockcollector

# Profiler
RUN apk update && apk add --no-cache curl

COPY ./image/bin/profiler /profiler
COPY ./profiler/pprof.sh /pprof.sh
COPY ./profiler/entrypoint.sh /profiler-entrypoint.sh
