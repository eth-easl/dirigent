#!/bin/bash

function startMonitoring() {
    ssh $1 "tmux kill-session -t resource_monitoring"
    ssh $1 "tmux new-session -d -s resource_monitoring"
    ssh $1 "tmux send-keys -t resource_monitoring 'python3 ~/cluster_manager/cmd/monitoring/monitoring.py' ENTER"
}

for ip in "$@"
do
    startMonitoring $ip &
done

wait