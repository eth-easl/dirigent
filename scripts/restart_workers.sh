#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


function RestartWorkers() {
    function internal_setup() {
        RemoteExec $1 "tmux kill-session -t worker"
        RemoteExec $1 "tmux new -s worker -d"

        RemoteExec $1 "sudo sysctl -w net.ipv4.conf.all.route_localnet=1"
        RemoteExec $1 "sudo iptables -t nat -F"

        RemoteExec $1 "sudo cp ~/cluster_manager/configs/firecracker/vmlinux-4.14.bin /cluster_manager/configs/firecracker/vmlinux-4.14.bin"
        RemoteExec $1 "sudo cp ~/cluster_manager/configs/firecracker/rootfs.ext4 /cluster_manager/configs/firecracker/rootfs.ext4"

        CMD=$"cd ~/cluster_manager/cmd/worker_node;git pull; git lfs pull ;git reset --hard origin/current; sudo env 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/usr/local/go/bin:/usr/local/bin/firecracker:/usr/local/bin/firecracker' /usr/local/go/bin/go run main.go --config config_cluster.yaml"

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


# StopWorkers $@
RestartWorkers $(python3 string.py --type worker-ha)

# sudo env 'PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/games:/usr/local/games:/snap/bin:/usr/local/go/bin:/usr/local/bin/firecracker:/usr/local/bin/firecracker' /usr/local/go/bin/go run main.go --config config_cluster.yaml
# rsync -av samples Francois@pc704.emulab.net:invitro/