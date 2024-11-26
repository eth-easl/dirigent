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