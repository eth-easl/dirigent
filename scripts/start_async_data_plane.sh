#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

RemoteExec $DATAPLANE "cd ~/cluster_manager/cmd/data_plane; git pull; git reset --hard origin/current;/usr/local/go/bin/go run main.go --config config_cluster_async.yaml"