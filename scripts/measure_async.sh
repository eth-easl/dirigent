#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


function RunMeasure() {
      RemoteExec $DATAPLANE "cd ~/cluster_manager/async;git pull; git reset --hard origin/current; /usr/local/go/bin/go run fire.go"

      scp $DATAPLANE:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$1.csv
      scp $CONTROLPLANE:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$1.csv
}


RunMeasure

# ./run.sh 1 2 4 8 16 32 50 100 200 400 800