# Produces an image containing a vulnerable version of sqlite-jdbc
#
# Created to validate the detection of CVE-2023-32697 by ACS Scanner.
#
# Pushed to: "quay.io/rhacs-eng/qa:sqlite-jdbc-CVE-2023-32697".

ARG URL=https://repo1.maven.org/maven2/org/xerial/sqlite-jdbc/3.41.2.1/sqlite-jdbc-3.41.2.1.jar
ARG OUTFILE=sqlite-jdbc-3.41.2.1.jar

# ==============

FROM registry.access.redhat.com/ubi9/ubi:latest AS builder
ARG URL
ARG OUTFILE

RUN dnf install -y wget

RUN mkdir /app && wget -q ${URL} -O /app/${OUTFILE} && chmod 755 /app/${OUTFILE}

# ==============

FROM registry.access.redhat.com/ubi9/ubi-minimal:latest
ARG OUTFILE

COPY --from=builder /app/${OUTFILE} /${OUTFILE}

CMD /bin/bash -c "trap : TERM INT; sleep infinity & wait"
