#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

RemoteExec $CONTROLPLANE "cd ~/cluster_manager/cmd/master_node; git checkout master; /usr/local/go/bin/go run main.go --configPath config_cluster.yaml"