#!/bin/bash

#
# MIT License
#
# Copyright (c) 2024 EASL
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

source $DIR/common.sh
source $DIR/setup.cfg

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
    RemoteExec $1 'if [ ! -d ~/cluster_manager ];then git clone https://github.com/eth-easl/dirigent.git cluster_manager; fi'
    RemoteExec $1 "bash ~/cluster_manager/scripts/setup_node.sh $2 $WORKER_RUNTIME"
    # LFS pull for VM kernel image and rootfs
    RemoteExec $1 'cd ~/cluster_manager; git pull; git lfs pull'
    RemoteExec $1 'sudo cp -r ~/cluster_manager/ /cluster_manager'
    RemoteExec $1 'git clone https://github.com/vhive-serverless/invitro --branch=rps_mode'
}

git lfs pull

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
