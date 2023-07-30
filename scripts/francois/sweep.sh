function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

readonly DATAPLANE=Francois@amd204.utah.cloudlab.us

function sweep() {
  RemoteExec $DATAPLANE "cd ~/cluster_manager/tests/sweep; sudo /usr/local/go/bin/go run main.go --frequency $SIZE --duration 60"
  scp Francois@amd204.utah.cloudlab.us:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$INVOCATIONS.csv
  scp Francois@amd195.utah.cloudlab.us:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$INVOCATIONS.csv
}

SIZE=$1
sweep
