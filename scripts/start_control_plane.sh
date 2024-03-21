#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

function StartControlplane() {

  #RemoteExec $1 "sudo apt update && sudo apt install -y haproxy"
  #RemoteExec $1 "sudo cp ~/cluster_manager/configs/haproxy.cfg /etc/haproxy/haproxy.cfg"

  # Kill old process
  RemoteExec $1 "sudo kill -9 \$(sudo lsof -t -i:9091)"

  # Start new data plane
  RemoteExec $1 "cd ~/cluster_manager/cmd/master_node; git pull; git reset --hard origin/current; sudo /usr/local/go/bin/go run main.go --config $2"
}

if [ "$HA" = true ] ;
then
  StartControlplane $CONTROLPLANE_1 config_cluster_raft_1.yaml &
  StartControlplane $CONTROLPLANE_2 config_cluster_raft_2.yaml &
  StartControlplane $CONTROLPLANE_3 config_cluster_raft_3.yaml
else
  StartControlplane $CONTROLPLANE config_cluster.yaml
fi