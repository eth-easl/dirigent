#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

SUFFIX=500
DESTINATION_DIR=/home/lcvetkovic/Desktop/dirigent_rps_sweep/d430_100_percent_cold_start

LOADER_NODE=cvetkovi@pc848.emulab.net
CONTROL_PLANE_NODE=cvetkovi@pc784.emulab.net
DATA_PLANE_NODE=cvetkovi@pc735.emulab.net

scp $LOADER_NODE:/users/cvetkovi/invitro/data/out/experiment_duration_5.csv ${DESTINATION_DIR}/rps_${SUFFIX}.csv

RemoteExec $CONTROL_PLANE_NODE "sudo cp /data/cold_start_trace.csv ~/cold_start_trace.csv"
scp $CONTROL_PLANE_NODE:~/cold_start_trace.csv ${DESTINATION_DIR}/cold_start_trace_${SUFFIX}.csv

RemoteExec $DATA_PLANE_NODE "sudo cp /data/proxy_trace.csv ~/proxy_trace.csv"
scp $DATA_PLANE_NODE:~/proxy_trace.csv ${DESTINATION_DIR}/proxy_trace_${SUFFIX}.csv