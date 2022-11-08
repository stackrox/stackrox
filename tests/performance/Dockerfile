FROM grafana/k6

# k6 specific EVN varuables
ENV VUS="1"
ENV ITERATIONS="10"
ENV DURATION="2h"

# Test specific EVN variables
ENV HOST=central.stackrox:443
ENV ROX_ADMIN_PASSWORD=pass

WORKDIR /performance

COPY ./tests ./tests
COPY ./src ./src

ENTRYPOINT [ "/bin/sh", "-c" ]

# Test will be executed and image will continue to run. Useful when deployed in K8s.
CMD [ "k6 run tests/testSacScopes.js --vus ${VUS} --iterations ${ITERATIONS} --duration ${DURATION} --out csv=/tmp/k6-test-result.csv && cat /tmp/k6-test-result.csv && tail -f /dev/null" ]
