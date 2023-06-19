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
    RemoteExec $CONTROL_PLANE "cd ~/cluster_manager/data_plane; git pull"
    RemoteExec $CONTROL_PLANE "tmux kill-session -t control_plane"
    RemoteExec $CONTROL_PLANE "tmux new -s control_plane -d"

    ARGS="${VERBOSITY}"
    CMD="cd ~/cluster_manager/data_plane; go run cmd/master_node/main.go ${ARGS}"
    RemoteExec $CONTROL_PLANE "tmux send -t control_plane \"$CMD\" ENTER"
}

function SetupDataPlane() {
    RemoteExec $DATA_PLANE "cd ~/cluster_manager/data_plane; git pull"
    RemoteExec $DATA_PLANE "tmux kill-session -t data_plane"
    RemoteExec $DATA_PLANE "tmux new -s data_plane -d"

    ARGS="--controlPlaneIP ${CP_IP_ADDRESS} ${VERBOSITY}"
    CMD="cd ~/cluster_manager/data_plane; go run cmd/data_plane/main.go ${ARGS}"
    RemoteExec $DATA_PLANE "tmux send -t data_plane \"$CMD\" ENTER"
}

function SetupWorkerNodes() {
    CNI_CONFIG_PATH="/etc/cni/config"
    ARGS="--controlPlaneIP ${CP_IP_ADDRESS} ${VERBOSITY} --cniConfigPath ${CNI_CONFIG_PATH}"
    CMD="cd ~/cluster_manager/data_plane; sudo /usr/local/go/bin/go run cmd/worker_node/main.go ${ARGS}"

    function internal_setup() {
        RemoteExec $1 "cd ~/cluster_manager/data_plane; git pull"
        RemoteExec $1 "tmux kill-session -t worker_daemon"
        RemoteExec $1 "tmux new -s worker_daemon -d"

        RemoteExec $1 "sudo cp ~/cluster_manager/data_plane/configs/cni.conf ${CNI_CONFIG_PATH}"
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
