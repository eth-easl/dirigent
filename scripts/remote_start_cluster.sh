#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

# Extracting control plane and data plane nodes and worker nodes
readonly CONTROL_PLANE=$1
shift
readonly DATA_PLANE=$1
shift

readonly VERBOSITY="--verbosity debug"
readonly CP_IP_ADDRESS=$(RemoteExec $CONTROL_PLANE "netstat -ie | grep -B1 '10.0.1' | sed -n 2p | tr -s ' ' | cut -d ' ' -f 3")
echo "Control plane IP is ${CP_IP_ADDRESS}"

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
    ARGS="--configPath cmd/worker_node/config_cluster.yaml"
    CMD="cd ~/cluster_manager; sudo env 'PATH=\$PATH:/usr/local/bin/firecracker' /usr/local/go/bin/go run cmd/worker_node/main.go ${ARGS}"

    function internal_setup() {
        # LFS pull for VM kernel image and rootfs
        RemoteExec $1 "cd ~/cluster_manager; git pull; git lfs pull"

        # Compile worker node daemon
        RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/worker_node"
        RemoteExec $1 "cd ~/cluster_manager/cmd/worker_node/; /usr/local/go/bin/go build main.go"
        RemoteExec $1 "sudo cp -r ~/cluster_manager/* /cluster_manager"

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
