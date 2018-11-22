FROM        debian:jessie
MAINTAINER  stigt

ENV container docker

RUN apt-get update && apt-get install -y \
    arping \
    apt-utils \
    hping3 \
    iputils-ping \
    iperf3 \
    iproute \
    libc-ares2 \
    libjson-c2 \
    logrotate \
    supervisor \    
    telnet \
    tcpdump \    
    traceroute

WORKDIR /

ADD         startup.sh /root/startup.sh

ADD         supervisord.conf /etc/supervisord.conf

# add Free Range Routing debian
ADD frr_6.0-1.debian8+1_amd64.deb /frr.deb

# install ffr
RUN dpkg -i frr.deb

RUN rm -f /frr.deb

# Configuration files
VOLUME /etc/frr

# Command
CMD ["/root/startup.sh"]
