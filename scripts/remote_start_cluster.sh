#!/bin/bash

#
# MIT License
#
# Copyright (c) 2024 EASL
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in all
# copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
# SOFTWARE.
#

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
    if [ "$CONTROL_PLANE_REPLICAS" -ne 1 ]; then
        CP_PREFIX="_raft"
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
