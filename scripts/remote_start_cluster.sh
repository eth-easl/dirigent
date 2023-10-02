#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
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
    RemoteExec $CONTROL_PLANE "tmux kill-session -t control_plane"
    RemoteExec $CONTROL_PLANE "tmux new -s control_plane -d"

    RemoteExec $CONTROL_PLANE "docker stop \$(docker ps -aq)"
    RemoteExec $CONTROL_PLANE "docker rm \$(docker ps -a -q)"
    RemoteExec $CONTROL_PLANE "docker run -d --name redis-stack-server -p 6379:6379 redis/redis-stack-server:latest"

    ARGS="--configPath cmd/master_node/config_cluster.yaml"
    CMD="cd ~/cluster_manager; go run cmd/master_node/main.go ${ARGS}"
    RemoteExec $CONTROL_PLANE "tmux send -t control_plane \"$CMD\" ENTER"
}

function SetupDataPlane() {
    RemoteExec $DATA_PLANE "cd ~/cluster_manager; git pull"

    IP_ADDRESS=$(RemoteExec $DATA_PLANE "netstat -ie | grep -B1 '10.0.1' | sed -n 2p | tr -s ' ' | cut -d ' ' -f 3")
    RemoteExec $DATA_PLANE 'export DATA_PLANE_IP=\"'$IP_ADDRESS'\"; cd cluster_manager; cat cmd/data_plane/config_cluster.yaml | envsubst > cmd/data_plane/tmp && mv cmd/data_plane/tmp cmd/data_plane/config_cluster.yaml'

    RemoteExec $DATA_PLANE "tmux kill-session -t data_plane"
    RemoteExec $DATA_PLANE "tmux new -s data_plane -d"

    ARGS="--configPath cmd/data_plane/config_cluster.yaml"
    CMD="cd ~/cluster_manager; go run cmd/data_plane/main.go ${ARGS}"
    RemoteExec $DATA_PLANE "tmux send -t data_plane \"$CMD\" ENTER"
}

function SetupWorkerNodes() {
    ARGS="--configPath cmd/worker_node/config_cluster.yaml"
    CMD="cd ~/cluster_manager; sudo env 'PATH=\$PATH:/usr/local/bin/firecracker' /usr/local/go/bin/go run cmd/worker_node/main.go ${ARGS}"

    function internal_setup() {
        # LFS pull for VM kernel image and rootfs
        RemoteExec $1 "cd ~/cluster_manager; git pull; git lfs pull"

        IP_ADDRESS=$(RemoteExec $1 "netstat -ie | grep -B1 '10.0.1' | sed -n 2p | tr -s ' ' | cut -d ' ' -f 3")
        RemoteExec $1 'export WORKER_NODE_IP=\"'$IP_ADDRESS'\"; cd cluster_manager; cat cmd/worker_node/config_cluster.yaml | envsubst > cmd/worker_node/tmp && mv cmd/worker_node/tmp cmd/worker_node/config_cluster.yaml'

        RemoteExec $1 "tmux kill-session -t worker_daemon"
        RemoteExec $1 "tmux new -s worker_daemon -d"

        RemoteExec $1 "tmux send -t worker_daemon \"$CMD\" ENTER"
    }

    for NODE in "$@"
    do
        internal_setup $NODE &
    done

    wait
}

# Starting processes
SetupControlPlane
SetupDataPlane
SetupWorkerNodes $@
