#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

source $DIR/common.sh
source $DIR/setup.cfg

function GenerateSSHKeys() {
    if [ ! -f "$DIR/ssh-dirigent" ]
    then
        ssh-keygen -t ed25519 -f "$DIR/ssh-dirigent" -N ''
        echo -e "${color_red}ACTION REQUIRED:${color_white} Please add a new SSH key to your GitHub profile."
        echo -e "${color_green}Step 1:${color_reset} Visit ${color_cyan}https://github.com/settings/ssh/new${color_reset}"
        echo -e "${color_green}Step 2:${color_reset} Paste the following contents:"
        cat "$DIR/ssh-dirigent.pub"
        read -p "Press Enter after making these changes."
    fi
}

function AddSshKeys() {
    exists=$(RemoteExec $1 'if [ -f "~/.ssh/id_rsa" ]; then echo "exists"; fi')

    if [ "$exists" != "exists" ]; then
        RemoteExec $1 'ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts'
        CopyToRemote "$DIR/ssh-dirigent" "$1:~/.ssh/id_rsa"
    fi
}

function SetupNode() {
    AddSshKeys $1
    RemoteExec $1 'if [ ! -d ~/cluster_manager ];then git clone --branch=sosp_24_submission git@github.com:eth-easl/cluster_manager.git; fi'
    RemoteExec $1 "bash ~/cluster_manager/scripts/setup_node.sh $2"
    # LFS pull for VM kernel image and rootfs
    RemoteExec $1 'cd ~/cluster_manager; git pull; git lfs pull'
    RemoteExec $1 'sudo cp -r ~/cluster_manager/ /cluster_manager'
    RemoteExec $1 'git clone https://github.com/vhive-serverless/invitro'
    CopyToRemote "$DIR/invitro_traces" "$1:invitro"
}

git lfs pull
python3 invitro_traces/generate_traces.py

GenerateSSHKeys

NODE_COUNTER=0

for NODE in "$@"
do
    if [ "$NODE_COUNTER" -eq 0 ]; then
        HA_SETTING="REDIS"
    elif [ "$NODE_COUNTER" -le $CONTROL_PLANE_REPLICAS ]; then
        HA_SETTING="CONTROL_PLANE"
    elif [ "$NODE_COUNTER" -le $(( $CONTROL_PLANE_REPLICAS + $DATA_PLANE_REPLICAS )) ]; then
        HA_SETTING="DATA_PLANE"
    else
        HA_SETTING="WORKER_NODE"
    fi

    SetupNode $NODE $HA_SETTING &
    let NODE_COUNTER++
done

wait
