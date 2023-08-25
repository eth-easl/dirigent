#!/bin/bash

function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

readonly DATAPLANE=Francois@hp146.utah.cloudlab.us
readonly CONTROLPLANE=Francois@hp155.utah.cloudlab.us
