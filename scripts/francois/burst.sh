#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

function Burst() {
  RemoteExec $DATAPLANE "cd ~/cluster_manager/tests/burst; sudo /usr/local/go/bin/go run main.go --invocations $INVOCATIONS"

  scp $DATAPLANE:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$INVOCATIONS.csv
  scp $CONTROLPLANE:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$INVOCATIONS.csv
}

INVOCATIONS=$1
Burst

