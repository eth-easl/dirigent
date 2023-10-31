#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

function AddSshKeys() {
    ACCESS_TOKEN="$(cat ~/.git_token_loader)"

    exists=$(RemoteExec $1 'if [ -f "~/.ssh/id_rsa" ]; then echo "exists"; fi')

    if [ "$exists" != "exists" ]; then
        RemoteExec $1 'echo -en "\n\n" | ssh-keygen -t rsa'
        RemoteExec $1 'ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts'
        RemoteExec $1 'curl -H "Authorization: token '"$ACCESS_TOKEN"'" --data "{\"title\":\"'"key:\$(hostname)"'\",\"key\":\"'"\$(cat ~/.ssh/id_rsa.pub)"'\"}" https://api.github.com/user/keys'
    fi
}

function SetupNode() {
    AddSshKeys $1
    RemoteExec $1 'if [ ! -d ~/cluster_manager ];then git clone git@github.com:eth-easl/cluster_manager.git; fi'
    RemoteExec $1 'bash ~/cluster_manager/scripts/setup_node.sh'
    RemoteExec $1 'cd ~/cluster_manager/cmd/master_node/; /usr/local/go/bin/go build main.go'
    RemoteExec $1 'cd ~/cluster_manager/cmd/data_plane/; /usr/local/go/bin/go build main.go'
    RemoteExec $1 'cd ~/cluster_manager/cmd/worker_node/; /usr/local/go/bin/go build main.go'
    RemoteExec $1 'sudo cp -r ~/cluster_manager /'
    RemoteExec $1 'sudo cp /cluster_manager/scripts/systemd/control_plane.service /etc/systemd/system/control_plane.service'
    RemoteExec $1 'sudo cp -r /cluster_manager/scripts/systemd/data_plane.service /etc/systemd/system/data_plane.service'
    RemoteExec $1 'sudo cp -r /cluster_manager/scripts/systemd/worker_node.service /etc/systemd/system/worker_node.service'
}

for NODE in "$@"
do
    SetupNode $NODE &
done

wait
