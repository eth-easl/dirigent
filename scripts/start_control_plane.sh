#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

RemoteExec $CONTROLPLANE "cd ~/cluster_manager/cmd/master_node; git checkout fix-concurrency; /usr/local/go/bin/go run main.go --config config_cluster.yaml"