#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

function Run() {
  RemoteExec $INVITRO "cd ~/invitro;git checkout rps_mode; sudo /usr/local/go/bin/go run cmd/loader.go --config ~/invitro/rps/$1/config_firecracker.json --verbosity trace"
  StoreResults $1
}

for VALUE in "$@"
do
  Run $VALUE
done