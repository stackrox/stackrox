FROM registry.fedoraproject.org/fedora:41

RUN dnf install -y which iproute bpftool procps iptables nc

COPY prepare-tap.sh /scripts/
