FROM debian:stretch AS compile-image
MAINTAINER stigt

RUN apt-get update && apt-get install -y --no-install-recommends wget build-essential flex bison libncurses-dev libreadline-dev git

WORKDIR /root
RUN wget ftp://bird.network.cz/pub/bird/bird-2.0.7.tar.gz
RUN tar zxvf bird-2.0.7.tar.gz
WORKDIR /root/bird-2.0.7
RUN ./configure --prefix=/usr --sysconfdir=/etc --runstatedir=/run
RUN make

FROM debian:stretch

RUN apt-get update && apt-get install -y --no-install-recommends \
	arping \
	wget \
	tcpdump \
	hping3 \
	procps \
	iproute \
	iputils-ping \
	net-tools \
	traceroute \
	supervisor \
	vim

WORKDIR /root
ADD startup.sh /root/startup.sh
ADD supervisord.conf /etc/supervisord.conf
COPY --from=compile-image /root/bird-2.0.7/bird .
COPY --from=compile-image /root/bird-2.0.7/birdc .
COPY --from=compile-image /root/bird-2.0.7/birdcl .

VOLUME /etc/bird

# Expose ports
EXPOSE 179 2601 2604 2605 5201

# Command
CMD ["/root/startup.sh"]
