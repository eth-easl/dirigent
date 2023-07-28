function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

readonly DATAPLANE=$1
shift

INVOCATIONS=800

RemoteExec $DATAPLANE "cd ~/cluster_manager/tests/burst; sudo /usr/local/go/bin/go run main.go --invocations $INVOCATIONS"
scp Francois@amd001.utah.cloudlab.us:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_800.csv
scp Francois@amd021.utah.cloudlab.us:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_800.csv
