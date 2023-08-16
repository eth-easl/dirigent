function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

readonly DATAPLANE=Francois@amd200.utah.cloudlab.us

function Burst() {
  RemoteExec $DATAPLANE "cd ~/cluster_manager/cmd/cleaner; sudo /usr/local/go/bin/go run cleaner.go"
  RemoteExec $DATAPLANE "cd ~/cluster_manager/tests/burst; sudo /usr/local/go/bin/go run main.go --invocations $INVOCATIONS"
  scp Francois@amd200.utah.cloudlab.us:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv proxy_trace_$INVOCATIONS.csv
  scp Francois@amd253.utah.cloudlab.us:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv cold_start_trace_$INVOCATIONS.csv
}

INVOCATIONS=$1
Burst
