#!/bin/bash

function internal() {
    file_name="$2/utilization_$1.csv"
    scp $1:~/utilization.csv $file_name
    ssh $1 "tmux kill-session -t resource_monitoring"
}

PREFIX=$1
shift

for ip in "$@"
do
    internal $ip $PREFIX &
done

wait