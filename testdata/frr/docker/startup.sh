#!/bin/bash

chown -R frr:frr /etc/frr
chmod 644 /etc/frr/*

service frr start

sleep infinity
