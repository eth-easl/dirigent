#!/bin/bash

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
for $NODE in "$@"
do
    AddSshKeys $NODE
    RemoteExec $NODE 'git clone https://github.com/eth-easl/cluster_manager.git'
    # install Golang on each worker node
done

# Extracting control plane and data plane nodes and worker nodes
CONTROL_PLANE=$1
shift
DATA_PLANE=$1
shift
WORKER_NODES=$1

function SetupControlPlane() {
    RemoteExec $CONTROL_PLANE 'cd ~/cluster_manager/data_plane; go run cmd/master_node/main.go'
}

function SetupDataPlane() {
    RemoteExec $DATA_PLANE 'cd ~/cluster_manager/data_plane; go run cmd/data_plane/main.go'
}

function SetupWorkerNodes() {
    # TODO: install docker on each worker node
    for $NODE in "$WORKER_NODES"
    do
        RemoteExec $NODE 'cd ~/cluster_manager/data_plane; go run cmd/worker_node/main.go'
    done
}

# Starting processes
SetupControlPlane &
SetupDataPlane &
SetupWorkerNodes &

wait