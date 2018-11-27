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
	docker-compose up -d

	ip link add dummy1 type dummy 2> /dev/null
	ip link add dummy2 type dummy 2> /dev/null
	
	$D_MOVE up CA-1 xeth10 10.1.0.1/24
	$D_MOVE up CA-1 dummy1 192.168.0.1/32
	
	$D_MOVE up RA-1 xeth2 10.1.0.2/24
	$D_MOVE up RA-1 xeth3 10.2.0.2/24
	
	$D_MOVE up RA-2 xeth4 10.2.0.3/24
	$D_MOVE up RA-2 xeth5 10.3.0.3/24
	
	$D_MOVE up CA-2 xeth6  10.3.0.4/24
	$D_MOVE up CA-2 dummy2 192.168.0.2/32

	;;
    "down")
	$D_MOVE down CA-1 xeth10
	$D_MOVE down RA-1 xeth2
	$D_MOVE down RA-1 xeth3
	$D_MOVE down RA-2 xeth4
	$D_MOVE down RA-2 xeth5 
	$D_MOVE down CA-2 xeth6

	ip link del dummy1
	ip link del dummy2	

	docker-compose down
	
	for ns in CA-1 RA-1 RA-2 CA-2 ; do
	   ip netn del $ns
	done
	chown -R $USER:$USER volumes
	;;
    *)
	echo "Unknown action"
	;;
esac
