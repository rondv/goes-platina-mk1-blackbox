FROM        debian:stretch
MAINTAINER  stigt

ENV container docker

RUN apt-get update && apt-get install -y \
    arping \
    apt-transport-https \
    apt-utils \
    ca-certificates \
    curl \
    gnupg2 \
    hping3 \
    iputils-ping \
    iperf3 \
    iproute \
    libc-ares2 \
    libjson-c3 \
    logrotate \
    lsb-release \
    procps \
    telnet \
    tcpdump \
    traceroute \
    vim

RUN curl -s https://deb.frrouting.org/frr/keys.asc | apt-key add -
RUN echo deb https://deb.frrouting.org/frr $(lsb_release -s -c) "frr-stable" | tee -a /etc/apt/sources.list.d/frr.list

RUN apt-get update && apt-get install -y \
    frr \
    frr-pythontools

WORKDIR /

ADD         startup.sh /root/startup.sh

# Configuration files
VOLUME /etc/frr

# Command
CMD ["/root/startup.sh"]
