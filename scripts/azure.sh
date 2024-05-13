#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

./start_resource_monitoring.sh $(python3 string.py)
function Run() {
  RemoteExec $INVITRO "cd ~/invitro;git reset --hard origin/rps_mode_predictive; sudo /usr/local/go/bin/go run cmd/loader.go  --config ~/invitro/invitro_traces/samples/$1/config.json --verbosity trace"

  scp $INVITRO:~/invitro/data/out/experiment_duration_5.csv plotting/azure_$1_5.csv
  scp $INVITRO:~/invitro/data/out/experiment_duration_180.csv plotting/azure_$1_180.csv
  scp $INVITRO:~/invitro/data/out/experiment_duration_120.csv plotting/azure_$1_120.csv

  scp $CONTROLPLANE:/users/Francois/cluster_manager/cmd/master_node/output_logs.txt plotting/logs_$1.txt
  #StoreResults $1
}

for VALUE in "$@"
do
  Run $VALUE
done


./collect_resource_monitoring.sh $(python3 string.py)