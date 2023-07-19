# Produces an image containing a vulnerable version of Java drools.
#
# Created to validate the detection of CVE-2021-41411 by ACS Scanner.
# 
# Pushed to: "quay.io/rhacs-eng/qa:drools-CVE-2021-41411".
FROM registry.access.redhat.com/ubi8/ubi:latest AS builder

ARG DROOLS_URL=https://repo1.maven.org/maven2/org/drools/drools-core/6.4.0.Final/drools-core-6.4.0.Final.jar 
ARG OUTFILE=org.drools.drools-core-6.4.0.Final.jar

RUN dnf install -y wget

RUN mkdir /app && wget -q ${DROOLS_URL} -O /app/${OUTFILE} && chmod 755 /app/${OUTFILE}

# =============

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.8

COPY --from=builder /app/${OUTFILE} /${OUTFILE}
