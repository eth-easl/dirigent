#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


function RunMeasure() {
      function Run() {
          RemoteExec $INVITRO "cd ~/invitro;git checkout rps_mode; sudo /usr/local/go/bin/go run cmd/loader.go  --config ~/invitro/samples/$1/config.json --verbosity trace"

          scp $INVITRO:~/invitro/data/out/experiment_duration_30.csv azure_$1.csv
      }

      for VALUE in "$@"
      do
          Run $VALUE
      done
}


RunMeasure $@

# ./run.sh 1 2 4 8 16 32 50 100 200 400 800