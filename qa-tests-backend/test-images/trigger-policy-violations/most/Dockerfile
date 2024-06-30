FROM registry.access.redhat.com/ubi8/ubi:8.2

WORKDIR /

# For: "Improper Usage of Orchestrator Secrets Volume"
# GJ 6/17/2020 - remove for openshift and see ROX-5098, this is not used
# RUN mkdir /run/secrets
# RUN echo "a secret" > /run/secrets/asecret
# VOLUME /run/secrets

# For: "Shell Spawned by Java Application"
RUN yum -y install csh

# Using ADD also triggers: "ADD Command used instead of COPY"
ADD trigger-violations.sh /trigger-violations-insecure.sh
COPY run-csh.sh /

# For: Secure Shell (ssh) Port Exposed in Image
EXPOSE 22/tcp

# For: Wget in Image
RUN yum -y install wget

# For: Curl in Image
RUN yum -y install curl

# For: Insecure specified in CMD
CMD ["/bin/bash", "-x", "trigger-violations-insecure.sh"]
