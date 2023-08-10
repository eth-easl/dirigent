function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

readonly DATA_PLANE=Francois@amd204.utah.cloudlab.us

function RunMeasure() {
      function Run() {
          RemoteExec $DATA_PLANE "cd ~/cluster_manager/cmd/cleaner; sudo /usr/local/go/bin/go run cleaner.go"
          RemoteExec $DATA_PLANE "cd loader; sudo /usr/local/go/bin/go run cmd/loader.go  --config config_$1.json --verbosity trace"

          scp Francois@amd204.utah.cloudlab.us:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$1.csv
          scp Francois@amd195.utah.cloudlab.us:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$1.csv
      }

      for VALUE in "$@"
      do
          Run $VALUE
      done
}


RunMeasure $@

# ./run.sh 1 2 4 8 16 32 50 100 200 400 800