# HEADER
FROM        debian:jessie
MAINTAINER  stigt

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    arping \
    hping3 \
    iputils-ping \
    iperf3 \
    iproute \
    net-tools \
    supervisor \    
    telnet \
    tcpdump \    
    traceroute \
    quagga \
    vim     

WORKDIR /root

ADD         startup.sh /root/startup.sh

ADD         supervisord.conf /etc/supervisord.conf

# add GO BGP 
ADD gobgpd /root/gobgpd
ADD gobgp  /root/gobgp

RUN useradd -r gobgpd

# Configuration files
VOLUME /etc/gobgp

# Expose ports
EXPOSE 179 2601 2604 2605 5201

# Command
CMD ["/root/startup.sh"]
