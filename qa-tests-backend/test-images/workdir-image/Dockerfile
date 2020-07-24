# ROX-4751 regression test - entrypoints with relative file path causes incorrect process path
# docker build -t stackrox/qa:ROX4751

FROM ubuntu
RUN mkdir -p test/bin && \
printf \
'#!/bin/bash    \n\
while true ; do \n\
  date          \n\
  sleep 33      \n\
done            \n'\
> /test/bin/exec.sh && \
chmod +x /test/bin/exec.sh

WORKDIR /test
CMD ["bin/exec.sh"]
