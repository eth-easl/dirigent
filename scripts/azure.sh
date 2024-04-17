#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

function Run() {
  RemoteExec $INVITRO "cd ~/invitro;git checkout rps_mode; sudo /usr/local/go/bin/go run cmd/loader.go  --config ~/invitro/samples/$1/config.json --verbosity trace"

  scp $INVITRO:~/invitro/data/out/experiment_duration_5.csv plotting/azure_$1.csv
  scp $INVITRO:~/invitro/data/out/experiment_duration_30.csv plotting/azure_$1.csv
  StoreResults $1
}

for VALUE in "$@"
do
  Run $VALUE
done
