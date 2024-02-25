#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

source $DIR/common.sh
source $DIR/setup.cfg

# REDIS
readonly REDIS_NODE=$1
shift
SetupRedis $REDIS_NODE

# Kill all Dirigent processes
KillSystemdServices $@

# Starting control plane(s)
for (( c=1; c<=$CONTROL_PLANE_REPLICAS; c++ ))
do
    if [ "$CONTROL_PLANE_REPLICAS" -eq 1 ]; then
        SetupControlPlane $1
    else
        SetupControlPlane $1 "_${c}"
    fi

    shift
done

# Starting control plane(s)
for (( c=1; c<=$DATA_PLANE_REPLICAS; c++ ))
do
    if [ "$DATA_PLANE_REPLICAS" -eq 1 ]; then
        SetupDataPlane $1
    else
        SetupDataPlane $1 "_${c}"
    fi

    shift
done

SetupWorkerNodes $@
