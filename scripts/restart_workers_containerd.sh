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


function RestartWorkers() {
    function internal_setup() {
        RemoteExec $1 "tmux kill-session -t worker"
        RemoteExec $1 "tmux new -s worker -d"

        RemoteExec $1 "sudo sysctl -w net.ipv4.conf.all.route_localnet=1"
        RemoteExec $1 "sudo iptables -t nat -F"

        CMD=$"cd ~/cluster_manager; git fetch origin;git reset --hard origin/current; sudo /usr/local/go/bin/go run cmd/worker_node/main.go --config cmd/worker_node/config_cluster_containerd.yaml"

        RemoteExec $1 "tmux send -t worker \"$CMD\" ENTER"
    }

    for NODE in "$@"
    do
        internal_setup $NODE &
    done


    wait
}

function StopWorkers() {
    function internal_setup() {
        RemoteExec $1 "tmux kill-session -t worker"
        #RemoteExec $1 "sudo kill -9 $(sudo lsof -t -i:10010)"
    }

    for NODE in "$@"
    do
        internal_setup $NODE &
    done

    wait
}


# StopWorkers $@
RestartWorkers $(python3 string.py --type worker-ha)

# sudo env 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/usr/local/go/bin:/usr/local/bin/firecracker:/usr/local/bin/firecracker' /usr/local/go/bin/go run main.go --config config_cluster.yaml
# rsync -av samples Francois@pc704.emulab.net:invitro/