#!/bin/bash

if ! [ $(id -u) = 0 ]; then
   echo "You must run this script as root/sudo."
   exit 1
fi
USER=$SUDO_USER

if [ -z "$1" ]; then
    echo "Usages: $0 up|down"
    exit 1
fi

D_MOVE=../docker_move.sh

case $1 in
    "up")
	docker-compose -f docker-compose_1base.yaml up -d
	ip link add dummy0 type dummy 2> /dev/null
	ip link add dummy1 type dummy 2> /dev/null
	ip link add dummy2 type dummy 2> /dev/null
	ip link add dummy3 type dummy 2> /dev/null

	$D_MOVE up R1 ${net1port0} 192.168.12.1/24
	$D_MOVE up R1 ${net0port1} 192.168.14.1/24
	$D_MOVE up R1 dummy0 192.168.1.1/32
	# extra dummy1 for route injection testing
	docker exec R1 ip link add dummy1 type dummy
	docker exec R1 ip link set dev dummy1 up

	$D_MOVE up R2 ${net1port1} 192.168.12.2/24
	$D_MOVE up R2 ${net2port1} 192.168.23.2/24
	$D_MOVE up R2 dummy1 192.168.2.2/32

	$D_MOVE up R3 ${net3port1} 192.168.34.3/24
	$D_MOVE up R3 ${net2port0} 192.168.23.3/24
	$D_MOVE up R3 dummy2 192.168.3.3/32

	$D_MOVE up R4 ${net3port0} 192.168.34.4/24
	$D_MOVE up R4 ${net0port0}  192.168.14.4/24
	$D_MOVE up R4 dummy3 192.168.4.4/32
	;;
    "down")
	$D_MOVE down R1 ${net1port0}
	$D_MOVE down R1 ${net0port1}
	$D_MOVE down R1 dummy0

	$D_MOVE down R2 ${net1port1}
	$D_MOVE down R2 ${net2port1}
	$D_MOVE down R2 dummy1

	$D_MOVE down R3 ${net3port1}
	$D_MOVE down R3 ${net2port0}
	$D_MOVE down R3 dummy2

	$D_MOVE down R4 ${net3port0}
	$D_MOVE down R4 ${net0port0}
	$D_MOVE down R4 dummy3

	docker-compose -f docker-compose_1base.yaml down
	
	for ns in R1 R2 R3 R4; do
	   ip netn del $ns
	done
	chown -R $USER:$USER volumes
	;;
    *)
	echo "Unknown action"
	;;
esac
