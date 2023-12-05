#!/bin/bash

function internal() {
    # Connect to the remote machine and start a tmux session
    ssh $1 "tmux new-session -d -s resource_monitoring"
    ssh $1 "tmux send-keys -t resource_monitoring 'python3 ~/cluster_manager/cmd/monitoring/monitoring.py' ENTER"
}

function install_infrastructure() {
    ssh $1 "sudo apt update && sudo apt install -y python3-pip && pip3 install psutil"
}

for ip in "$@"
do
    internal $ip &
done

wait