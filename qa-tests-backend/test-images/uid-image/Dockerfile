# ROX-4979 regression test - incorrect process UID when set via runAsUser or USER instruction
# docker build -t stackrox/qa:ROX4979 .
FROM alpine:3.20

RUN addgroup -S -g 9000 qa_group && \
    adduser -S -G qa_group -u 9001 qa_user && \
    mkdir /qa && \
    chown -R qa_user:qa_group /qa

RUN printf \
'#!/bin/sh      \n\
while true ; do \n\
  sleep 30      \n\
done            \n'\
> /qa/exec.sh && \
chown qa_user:qa_group /qa/exec.sh && \
chmod +x /qa/exec.sh

USER 9001:9000
ENTRYPOINT ["/qa/exec.sh"]
