#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh


echo -e "${color_cyan}Loading Invitro traces${color_reset}"
rsync -av invitro_traces/* $INVITRO:invitro/invitro_traces

echo -e "${color_cyan}Starting monitoring${color_reset}"
./start_resource_monitoring.sh $(python3 string.py)

echo -e "${color_cyan}Starting experiment${color_reset}"
RemoteExec $INVITRO "cd ~/invitro;git pull; git reset --hard origin/FRANCOIS; sudo /usr/local/go/bin/go run cmd/loader.go  --config ~/invitro/invitro_traces/day6_hour8_samples/$1/config.json --verbosity trace"

echo -e "${color_cyan}Downloading Invitro output${color_reset}"
# Try downloading multiple durations (5,30,120,180 are the most frequents
scp $INVITRO:~/invitro/data/out/experiment_duration_180.csv plotting/azure_$1_180.csv
scp $INVITRO:~/invitro/data/out/experiment_duration_120.csv plotting/azure_$1_120.csv
scp $INVITRO:~/invitro/data/out/experiment_duration_30.csv plotting/azure_$1_30.csv
scp $INVITRO:~/invitro/data/out/experiment_duration_5.csv plotting/azure_$1_5.csv

echo -e "${color_cyan}Downloading control plane output${color_reset}"
scp $CONTROLPLANE:~/cluster_manager/cmd/master_node/output_logs.txt plotting/logs_$1.txt

echo -e "${color_cyan}Collecting monitoring logs${color_reset}"
./collect_resource_monitoring.sh $(python3 string.py)