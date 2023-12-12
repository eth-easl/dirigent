#!/bin/bash

function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

readonly DATAPLANE=Francois@hp156.utah.cloudlab.us
readonly CONTROLPLANE=Francois@hp158.utah.cloudlab.us
