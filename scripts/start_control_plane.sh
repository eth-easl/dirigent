#!/bin/bash

#
# MIT License
#
# Copyright (c) 2024 EASL
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

function StartControlplane() {

  #RemoteExec $1 "sudo apt update && sudo apt install -y haproxy"
  #RemoteExec $1 "sudo cp ~/cluster_manager/configs/haproxy.cfg /etc/haproxy/haproxy.cfg"

  # Kill old process
  RemoteExec $1 "sudo kill -9 \$(sudo lsof -t -i:9091)"

  # Start new data plane
  #RemoteExec $1 "cd ~/cluster_manager; sudo docker-compose up -d"
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