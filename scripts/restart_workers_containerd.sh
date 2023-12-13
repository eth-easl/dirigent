#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


function RestartWorkers() {
    function internal_setup() {
        RemoteExec $1 "tmux kill-session -t worker"
        RemoteExec $1 "tmux new -s worker -d"

        RemoteExec $1 "sudo sysctl -w net.ipv4.conf.all.route_localnet=1"

        CMD=$"cd ~/cluster_manager; git fetch origin;git reset --hard origin/master2; cd ~/cluster_manager/; sudo /usr/local/go/bin/go run cmd/worker_node/main.go --config cmd/worker_node/config_cluster_containerd.yaml"

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


#StopWorkers $@
RestartWorkers $@

# sudo env 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/usr/local/go/bin:/usr/local/bin/firecracker:/usr/local/bin/firecracker' /usr/local/go/bin/go run main.go --config config_cluster.yaml
# rsync -av samples Francois@pc704.emulab.net:invitro/