FROM ubuntu:jammy

COPY open-close-ports-load.c /open-close-ports-load.c
COPY open-close-ports-load.sh /open-close-ports-load.sh

RUN apt update -y && apt install gcc lsof -y
RUN gcc open-close-ports-load.c -o open-close-ports-load
