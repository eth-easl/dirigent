#!/bin/bash

set -e

function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

function AddSshKeys() {
    ACCESS_TOKEN="$(cat ~/.git_token_loader)"

    RemoteExec $1 'echo -en "\n\n" | ssh-keygen -t rsa'
    RemoteExec $1 'ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts'
    RemoteExec $1 'curl -H "Authorization: token '"$ACCESS_TOKEN"'" --data "{\"title\":\"'"key:\$(hostname)"'\",\"key\":\"'"\$(cat ~/.ssh/id_rsa.pub)"'\"}" https://api.github.com/user/keys'
}

# Add SSH keys for cloning the cluster manager repo
for NODE in "$@"
do
    #AddSshKeys $NODE
    RemoteExec $NODE 'git clone --branch=cluster_setup git@github.com:eth-easl/cluster_manager.git'
    RemoteExec $NODE 'bash ~/cluster_manager/data_plane/scripts/setup_node.sh'
done

# Extracting control plane and data plane nodes and worker nodes
CONTROL_PLANE=$1
shift
DATA_PLANE=$1
shift
WORKER_NODES=$@

function SetupControlPlane() {
    ARGS=""
    RemoteExec $CONTROL_PLANE "cd ~/cluster_manager/data_plane; go run cmd/master_node/main.go ${ARGS}"
}

function SetupDataPlane() {
    ARGS=""
    RemoteExec $DATA_PLANE "cd ~/cluster_manager/data_plane; go run cmd/data_plane/main.go ${ARGS}"
}

function SetupWorkerNodes() {
    ARGS=""

    for NODE in "$WORKER_NODES"
    do
        RemoteExec $NODE "cd ~/cluster_manager/data_plane; go run cmd/worker_node/main.go ${ARGS}"
    done
}

# Starting processes
SetupControlPlane &
SetupDataPlane &
SetupWorkerNodes &

wait