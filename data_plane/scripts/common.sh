#!/bin/bash

function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}