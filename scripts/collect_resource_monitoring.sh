#!/bin/bash

function internal() {
    file_name="/home/lcvetkovic/Desktop/characterization/azure_200/without_sidecars/utilization_$1.csv"
    scp $1:~/utilization.csv $file_name
    ssh $1 "tmux kill-session -t resource_monitoring"
}

for ip in "$@"
do
    internal $ip &
done

wait