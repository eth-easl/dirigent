#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

RemoteExec $INVITRO "cd ~/cluster_manager/; sudo docker-compose up"