#!/bin/bash

function internal() {
    file_name="/home/lcvetkovic/Desktop/replay/sosp/dirigent_1cp_1dp/azure_4000_firecracker/cpu_mem_usage/utilization_$1.csv"
    scp $1:~/utilization.csv $file_name
    ssh $1 "tmux kill-session -t resource_monitoring"
}

for ip in "$@"
do
    internal $ip &
done

wait