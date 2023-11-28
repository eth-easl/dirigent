#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/common.sh

# Extracting control plane and data plane nodes and worker nodes
readonly CONTROL_PLANE=$1
shift
readonly DATA_PLANE=$1
shift

KillSystemdServices $CONTROL_PLANE $DATA_PLANE $@

# Starting processes
SetupControlPlane $CONTROL_PLANE
SetupDataPlane $DATA_PLANE
SetupWorkerNodes $CONTROL_PLANE $DATA_PLANE $@
