#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

export INTERFACE_NAME=$(ifconfig | grep -B1 "10.0.1" | head -n1 | sed 's/:.*//')

cat $DIR/keepalived.conf | envsubst > $DIR/keepalived.conff

echo "Successfully created HA load balancer configuration!"