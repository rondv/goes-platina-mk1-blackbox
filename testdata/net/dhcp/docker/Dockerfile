# HEADER
FROM        debian:buster
MAINTAINER  stigt

RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y \
    arping \
    isc-dhcp-client \    
    isc-dhcp-server \
    hping3 \
    init-system-helpers \
    iputils-ping \    
    iperf3 \
    iproute2 \
    libc-ares2 \
    libjson-c3 \
    logrotate \
    net-tools \
    procps \
    supervisor \    
    telnet \
    tcpdump \    
    traceroute\
    vim      

ADD         startup.sh /root/startup.sh

ADD         supervisord.conf /etc/supervisord.conf

ADD	    dhcpd_4.conf /root/dhcpd_4.conf

ADD	    dhcpd_6.conf /root/dhcpd_6.conf

RUN	    touch /root/dhcpd.leases

WORKDIR /root

# Command
CMD ["/root/startup.sh"]
