#!/bin/bash


function internal() {
    file_name="utilization_$1.csv"
    scp $1:~/cluster_manager/cmd/monitoring/utilization.csv $file_name
    ssh $1 "tmux kill-session -t resource_monitoring"
}

for ip in "$@"
do
    internal $ip &
done

wait