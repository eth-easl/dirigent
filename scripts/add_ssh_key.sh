#!/bin/bash

readonly KEY="cvetkovi@node-99.boze-pomozi.faas-sched-pg0.utah.cloudlab.us"

function internal() {
    # Connect to the remote machine and start a tmux session
    ssh $1 "echo $KEY >> ~/.ssh/authorized_keys"
}

for ip in "$@"
do
    internal $ip &
done

wait