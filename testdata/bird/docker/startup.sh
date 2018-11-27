#!/bin/bash

chown -R bird:bird /etc/bird
chmod 644 /etc/bird/*
mkdir -p /usr/local/var/run
exec /usr/bin/supervisord -c /etc/supervisord.conf
