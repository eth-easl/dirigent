function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}


readonly  INVOCATIONS=$1
shift

readonly DATAPLANE=Francois@ms1145.utah.cloudlab.us

RemoteExec $DATAPLANE "cd ~/cluster_manager/tests/burst; sudo /usr/local/go/bin/go run main.go --invocations $INVOCATIONS"
scp Francois@ms1145.utah.cloudlab.us:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$INVOCATIONS.csv
scp Francois@amd224.utah.cloudlab.us:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$INVOCATIONS.csv
