#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

RemoteExec $DATAPLANE "cd ~/cluster_manager/tests/burst; sudo /usr/local/go/bin/go run main.go --invocations 1"