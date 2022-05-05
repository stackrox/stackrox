FROM registry.access.redhat.com/ubi8/nodejs-16 AS builder

USER 0:0

WORKDIR /workspace
COPY content content
COPY tools tools
COPY Makefile ./

RUN npm install -g npm@8.1.4 && npm install -g yarn@1.22.17
RUN make

FROM nginx:alpine
COPY nginx.conf /etc/nginx/nginx.conf

# This is well known localtion that other Dockerfiles relies on.
# Do not change until you ensure that all places were updated.
WORKDIR /docs/public

COPY --from=builder /workspace/build/ .

EXPOSE 8080
