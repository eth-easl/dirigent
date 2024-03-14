#!/bin/bash

function internal() {
    # Connect to the remote machine and start a tmux session
    ssh $1 "tmux new-session -d -s resource_monitoring"
    ssh $1 "tmux send-keys -t resource_monitoring 'python3 ~/cluster_manager/cmd/monitoring/monitoring.py' ENTER"
}

for ip in "$@"
do
    internal $ip &
done

wait