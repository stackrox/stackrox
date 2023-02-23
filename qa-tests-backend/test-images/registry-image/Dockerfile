FROM debian:11.4

RUN apt-get -y update || true
RUN apt-get -y install libssh-4=0.9.5-1+deb11u1 wget=1.21-1+deb11u1 tomcat9=9.0.43-2~deb11u3 openjdk-11-jre-headless=11.0.15+10-1~deb11u1 libtinfo5=6.2+20201114-2 && \
    wget https://raw.githubusercontent.com/opsxcq/exploit-CVE-2014-6271/master/packages/bash_4.2%252Bdfsg-0.1_amd64.deb -O bash.deb && \
    dpkg -i bash.deb && \
    apt-get -y install bash=4.2+dfsg-0.1 && \
    rm -f bash.deb

RUN mkdir /usr/share/tomcat9/temp

ENV PATH /usr/share/tomcat9/bin:"$PATH"

# trigger 'ADD COMMAND USED INSTEAD OF COPY' violation
ADD tomcat-files/* /usr/share/tomcat9/conf/

RUN touch /visa

# trigger 'SECURE SHELL (SSH) PORT EXPOSED IN IMAGE' violation
EXPOSE 22/tcp

CMD ["catalina.sh", "run"]
