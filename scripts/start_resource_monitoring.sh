#!/bin/bash

function internal() {
    # Connect to the remote machine and start a tmux session
    scp /home/lcvetkovic/projects/cluster_manager/cmd/monitoring/monitoring.py $1:~/monitoring.py

    ssh $1 "tmux kill-session -t resource_monitoring"
    ssh $1 "tmux new-session -d -s resource_monitoring"
    ssh $1 "tmux send-keys -t resource_monitoring 'python3 ~/monitoring.py' ENTER"
}

for ip in "$@"
do
    internal $ip &
done

wait