#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

source $DIR/common.sh
source $DIR/setup.cfg

# Skip the loader node
shift

# Kill all Dirigent processes
KillSystemdServices $@

# Starting control plane(s)
for (( c=1; c<=$CONTROL_PLANE_REPLICAS; c++ ))
do
    if [ "$CONTROL_PLANE_REPLICAS" -eq 1 ]; then
        SetupControlPlane $1
    else
        SetupControlPlane $1 "_raft_${c}"
    fi

    shift
done

# Starting data plane(s)
for (( c=1; c<=$DATA_PLANE_REPLICAS; c++ ))
do
    CP_PREFIX=""
    if [ "$ASYNC_DP_MODE" -eq 1 ]; then
        CP_PREFIX="${CP_PREFIX}_async"
    fi
    if [ "$CONTROL_PLANE_REPLICAS" -ne 1 ]; then
        CP_PREFIX="${CP_PREFIX}_raft"
    fi

    SetupDataPlane $1 $CP_PREFIX

    shift
done


if [ "$FAKE_WORKER_MODE" -eq 1 ]; then
    KillFakeWorkerNodes $@
    SetupFakeWorkerNodes $CONTROL_PLANE_REPLICAS $@
else
    SetupWorkerNodes $CONTROL_PLANE_REPLICAS $@
fi
