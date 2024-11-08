FROM registry.access.redhat.com/ubi9:latest

WORKDIR /

# For: "Improper Usage of Orchestrator Secrets Volume"
# GJ 6/17/2020 - remove for openshift and see ROX-5098, this is not used
# RUN mkdir /run/secrets
# RUN echo "a secret" > /run/secrets/asecret
# VOLUME /run/secrets

RUN yum update -y

# Using ADD also triggers: "ADD Command used instead of COPY"
ADD trigger-violations.sh /trigger-violations-insecure.sh
COPY run-bash.sh /

# For: trigger-violations.sh to trigger systemctl violation
RUN yum update -y && yum install systemd -y

# For: Secure Shell (ssh) Port Exposed in Image
EXPOSE 22/tcp

# For: Wget in Image
RUN yum -y install wget

# For: chkconfig in Image
RUN yum -y install chkconfig

# For: Insecure specified in CMD
CMD ["/bin/bash", "-x", "trigger-violations-insecure.sh"]
