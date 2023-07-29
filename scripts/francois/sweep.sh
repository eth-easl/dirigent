function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

readonly DATAPLANE=Francois@ms1145.utah.cloudlab.us


function sweep() {
  RemoteExec $DATAPLANE "cd ~/cluster_manager/tests/sweep; sudo /usr/local/go/bin/go run main.go --frequency $SIZE --duration 60"
  scp Francois@ms1145.utah.cloudlab.us:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$SIZE.csv
  scp Francois@amd224.utah.cloudlab.us:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$SIZE.csv
}

SIZE=$1
sweep
