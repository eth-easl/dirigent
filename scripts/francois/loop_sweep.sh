#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


function RunMeasure() {
      function Burst() {
        RemoteExec $DATAPLANE "cd ~/cluster_manager/tools/cleaner; sudo /usr/local/go/bin/go run cleaner.go"
        RemoteExec $DATAPLANE "cd ~/cluster_manager/tools/sweep; sudo /usr/local/go/bin/go run main.go --frequency $1"

        scp $DATAPLANE:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$1.csv
        scp $CONTROLPLANE:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$1.csv
      }

      for VALUE in "$@"
      do
          Burst $VALUE

          sleep 15
      done
}


RunMeasure $@

# ./loop_sweep.sh 1 2 4 8 16 32 50 100 200 400 800