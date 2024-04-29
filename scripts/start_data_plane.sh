#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

function StartDataplane() {
  # Kill old process
  RemoteExec $1 "sudo kill -9 \$(sudo lsof -t -i:8081)"

  # Start new dataplane
  RemoteExec $1 "cd ~/cluster_manager/cmd/data_plane; git pull; git reset --hard origin/current;sudo /usr/local/go/bin/go run main.go --config config_cluster.yaml"
}

#RemoteExec $DATAPLANE "sudo rm -rf /usr/local/go"
#RemoteExec $DATAPLANE "wget --continue --quiet https://go.dev/dl/go1.22.2.linux-amd64.tar.gz"
#RemoteExec $DATAPLANE "sudo tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz"
#RemoteExec $DATAPLANE "export PATH=$PATH:/usr/local/go/bin"
#RemoteExec $DATAPLANE "sudo sh -c  \"echo 'export PATH=\$PATH:/usr/local/go/bin' >> /etc/profile\""


if $HA;
then
  StartDataplane $DATAPLANE_1 &
  StartDataplane $DATAPLANE_2 &
  StartDataplane $DATAPLANE_3
else
  StartDataplane $DATAPLANE
fi
