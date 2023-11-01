#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

# Extracting control plane and data plane nodes and worker nodes
readonly CONTROL_PLANE=$1
shift
readonly DATA_PLANE=$1
shift

function SetupControlPlane() {
    RemoteExec $CONTROL_PLANE "cd ~/cluster_manager; git pull"

    # Start Redis server
    RemoteExec $CONTROL_PLANE "docker stop \$(docker ps -aq)"
    RemoteExec $CONTROL_PLANE "docker rm \$(docker ps -a -q)"
    RemoteExec $CONTROL_PLANE "docker run -d --name redis-stack-server -p 6379:6379 redis/redis-stack-server:latest"

    # Compile control plane
    RemoteExec $CONTROL_PLANE "sudo mkdir -p /cluster_manager/cmd/master_node"
    RemoteExec $CONTROL_PLANE "cd ~/cluster_manager/cmd/master_node/; /usr/local/go/bin/go build main.go"
    RemoteExec $CONTROL_PLANE "sudo cp ~/cluster_manager/cmd/master_node/main /cluster_manager/cmd/master_node/"
    RemoteExec $CONTROL_PLANE "sudo cp ~/cluster_manager/cmd/master_node/config_cluster.yaml /cluster_manager/cmd/master_node/"

    # Start control plane
    RemoteExec $CONTROL_PLANE "sudo systemctl daemon-reload && sudo systemctl start control_plane.service"
}

function SetupDataPlane() {
    RemoteExec $DATA_PLANE "cd ~/cluster_manager; git pull"

    # Compile data plane
    RemoteExec $DATA_PLANE "sudo mkdir -p /cluster_manager/cmd/data_plane"
    RemoteExec $DATA_PLANE "cd ~/cluster_manager/cmd/data_plane/; /usr/local/go/bin/go build main.go"
    RemoteExec $DATA_PLANE "sudo cp ~/cluster_manager/cmd/data_plane/main /cluster_manager/cmd/data_plane/"
    RemoteExec $DATA_PLANE "sudo cp ~/cluster_manager/cmd/data_plane/config_cluster.yaml /cluster_manager/cmd/data_plane/"

    # Start data plane
    RemoteExec $DATA_PLANE "sudo systemctl daemon-reload && sudo systemctl start data_plane.service"
}

function SetupWorkerNodes() {
    function internal_setup() {
        # LFS pull for VM kernel image and rootfs
        RemoteExec $1 "cd ~/cluster_manager; git pull; git lfs pull"

        # Compile worker node daemon
        RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/worker_node"
        RemoteExec $1 "cd ~/cluster_manager/cmd/worker_node/; /usr/local/go/bin/go build main.go"
        RemoteExec $1 "sudo cp -r ~/cluster_manager/* /cluster_manager"

        # For readiness probe
        RemoteExec $1 "sudo sysctl -w net.ipv4.conf.all.route_localnet=1"

        # Start worker node daemon
        RemoteExec $1 "sudo systemctl daemon-reload && sudo systemctl start worker_node.service"
    }

    for NODE in "$@"
    do
        internal_setup $NODE &
    done

    wait
}

function KillSystemdServices() {
    function internal_kill() {
        RemoteExec $1 "sudo systemctl stop worker_node"
    }

    RemoteExec $CONTROL_PLANE "sudo systemctl stop control_plane"
    RemoteExec $DATA_PLANE "sudo systemctl stop data_plane"

    for NODE in "$@"
    do
        internal_kill $NODE &
    done

    wait
}

KillSystemdServices $@

# Starting processes
SetupControlPlane
SetupDataPlane
SetupWorkerNodes $@
