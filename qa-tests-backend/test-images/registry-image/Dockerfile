FROM debian:12.6

RUN apt-get -y update || true
RUN apt-get -y install libssh-4=0.6.3-4+deb8u2 wget=1.16-1+deb8u7 tomcat7=7.0.56-3+deb8u11 openjdk-7-jre-headless=7u261-2.6.22-1~deb8u1 && \
    wget https://raw.githubusercontent.com/opsxcq/exploit-CVE-2014-6271/master/packages/bash_4.2%252Bdfsg-0.1_amd64.deb -O bash.deb && \
    dpkg -i bash.deb && \
    apt-get -y install bash=4.2+dfsg-0.1 && \
    rm -f bash.deb

RUN mkdir /usr/share/tomcat7/temp

ENV PATH /usr/share/tomcat7/bin:"$PATH"

# trigger 'ADD COMMAND USED INSTEAD OF COPY' violation
ADD tomcat-files/* /usr/share/tomcat7/conf/

RUN touch /visa

# trigger 'SECURE SHELL (SSH) PORT EXPOSED IN IMAGE' violation
EXPOSE 22/tcp

CMD ["catalina.sh", "run"]
