#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


function RestartWorkers() {
    function internal_setup() {
        RemoteExec $1 "tmux kill-session -t worker"
        RemoteExec $1 "tmux new -s worker -d"

        CMD=$"cd ~/cluster_manager/cmd/worker_node;git pull;git reset --hard origin/master2; sudo env 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/usr/local/go/bin:/usr/local/bin/firecracker:/usr/local/bin/firecracker' /usr/local/go/bin/go run main.go --config config_cluster.yaml"
        RemoteExec $1 "sudo sysctl -w net.ipv4.conf.all.route_localnet=1"

        # CMD=$"cd ~/cluster_manager; git fetch origin; cd ~/cluster_manager/cmd/worker_node; sudo /usr/local/go/bin/go run main.go --config config_cluster.yaml"

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
        RemoteExec $1 "sudo rm -rf /tmp"
        RemoteExec $1 "sudo mkdir /tmp"
        RemoteExec $1 "sudo chmod 777 /tmp"
        RemoteExec $1 "tmux kill-session -t worker"
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