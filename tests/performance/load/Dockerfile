FROM grafana/k6

# k6 specific ENV variables
ENV VUS="1"
ENV ITERATIONS="10"
ENV DURATION="2h"

# Test specific ENV variables
ENV HOST=https://central.stackrox:443
ENV ROX_ADMIN_PASSWORD=pass
ENV TEST_FILE=tests/testSacScopes.js
ENV OUTPUT_CSV=/tmp/k6-test-result.csv

WORKDIR /performance

# Install bash, because it's used by docker-entrypoint.sh
USER root
RUN apk add --no-cache bash
USER 12345

# Order of directories is based on change probability
COPY ./scripts ./scripts
COPY ./src ./src
COPY ./tests ./tests
COPY ./groups ./groups

ENTRYPOINT [ "/bin/sh", "-c" ]

# Test will be executed and image will continue to run. Useful when deployed in K8s.
CMD [ "./scripts/docker-entrypoint.sh" ]
