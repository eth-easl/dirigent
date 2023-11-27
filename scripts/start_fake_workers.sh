#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

readonly DAEMONS_PER_NODE=1

function SetupWorkerNodes() {
    local ARGS="--configPath cmd/worker_node/config_cluster_fake_worker.yaml"
    local CMD="cd ~/cluster_manager; sudo env 'PATH=\$PATH:/usr/local/bin/firecracker' /usr/local/go/bin/go run cmd/worker_node/main.go ${ARGS}"

    function internal_setup() {
        RemoteExec $1 "cd ~/cluster_manager; git pull; git lfs pull"

        for INDEX in `seq 0 $(($DAEMONS_PER_NODE - 1))`;
        do
            local DAEMON_PORT=$((10010 + $INDEX))

            RemoteExec $1 'export DAEMON_PORT=\"'$DAEMON_PORT'\"; cd cluster_manager; cat cmd/worker_node/config_cluster.yaml | envsubst > cmd/worker_node/tmp && mv cmd/worker_node/tmp cmd/worker_node/config_cluster.yaml'

            RemoteExec $1 "tmux new -s worker_daemon_${INDEX} -d"
            RemoteExec $1 "tmux send -t worker_daemon_${INDEX} \"$CMD\" ENTER"
        done
    }

    for NODE in "$@"
    do
        internal_setup $NODE &
    done
    wait
}

function KillWorkerNodes() {
    function internal_kill() {
        local PID_TO_KILL=$(RemoteExec $1 "ps -aux | grep cmd/worker_node | awk '{print \$2}' | tr '\n' ' '")


        for INDEX in `seq 0 $(($DAEMONS_PER_NODE - 1))`;
        do
            RemoteExec $1 "sudo kill -9 ${PID_TO_KILL}"
            RemoteExec $1 "tmux kill-session -t worker_daemon_${INDEX}"
        done
    }

    for NODE in "$@"
    do
        internal_kill $NODE &
    done

    wait
}

readonly CONTROL_PLANE=$1
shift
readonly DATA_PLANE=$1
shift

KillSystemdServices $CONTROL_PLANE $DATA_PLANE
KillWorkerNodes $@

SetupControlPlane $CONTROL_PLANE
SetupDataPlane $DATA_PLANE

SetupWorkerNodes $@