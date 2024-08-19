#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

function internal() {
    # Connect to the remote machine and start a tmux session
    scp $DIR/../cmd/monitoring/monitoring.py $1:~/monitoring.py

    ssh $1 "tmux kill-session -t resource_monitoring"
    ssh $1 "tmux new-session -d -s resource_monitoring"
    ssh $1 "tmux send-keys -t resource_monitoring 'python3 ~/monitoring.py' ENTER"
}

for ip in "$@"
do
    internal $ip &
done

wait