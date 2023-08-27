#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


function RunMeasure() {
      function Run() {
          RemoteExec $DATAPLANE "cd ~/cluster_manager/tools/cleaner; sudo /usr/local/go/bin/go run cleaner.go"
          RemoteExec $DATAPLANE "cd loader; sudo /usr/local/go/bin/go run cmd/loader.go  --config config_$1.json --verbosity trace"

          scp $DATAPLANE:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$1.csv
          scp $CONTROLPLANE:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$1.csv
      }

      for VALUE in "$@"
      do
          Run $VALUE
      done
}


RunMeasure $@

# ./run.sh 1 2 4 8 16 32 50 100 200 400 800