#!/bin/bash

readonly INVITRO=Francois@hp080.utah.cloudlab.us

readonly CONTROLPLANE=Francois@hp091.utah.cloudlab.us

readonly CONTROLPLANE_1=Francois@hp091.utah.cloudlab.us
readonly CONTROLPLANE_2=Francois@hp081.utah.cloudlab.us
readonly CONTROLPLANE_3=Francois@hp023.utah.cloudlab.us

readonly DATAPLANE=Francois@hp081.utah.cloudlab.us

readonly DATAPLANE_1=Francois@hp149.utah.cloudlab.us
readonly DATAPLANE_2=Francois@hp077.utah.cloudlab.us
readonly DATAPLANE_3=Francois@hp134.utah.cloudlab.us

readonly HA=false

function RemoteExec() {
    ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

function SetupControlPlane() {
    # Start Redis server
    RemoteExec $1 "sudo docker stop \$(docker ps -aq)"
    RemoteExec $1 "sudo docker rm \$(docker ps -a -q)"
    RemoteExec $1 "sudo docker run -d --name redis-stack-server -p 6379:6379 redis/redis-stack-server:latest"

    RemoteExec $1 "cd ~/cluster_manager; git pull"

    # Compile control plane
    RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/master_node"
    RemoteExec $1 "cd ~/cluster_manager/cmd/master_node/; /usr/local/go/bin/go build main.go"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/master_node/main /cluster_manager/cmd/master_node/"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/master_node/config_cluster$2.yaml /cluster_manager/cmd/master_node/config_cluster.yaml"

    # Remove old logs
    RemoteExec $1 "sudo journalctl --vacuum-time=1s && sudo journalctl --vacuum-time=1d"
    # Update systemd
    RemoteExec $1 "sudo cp -a ~/cluster_manager/scripts/systemd/* /etc/systemd/system/"
    # Start control plane
    RemoteExec $1 "sudo systemctl daemon-reload && sudo systemctl restart control_plane.service"
}

function SetupDataPlane() {
    RemoteExec $1 "cd ~/cluster_manager; git pull"

    # Compile data plane
    RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/data_plane"
    RemoteExec $1 "cd ~/cluster_manager/cmd/data_plane/; /usr/local/go/bin/go build main.go"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/data_plane/main /cluster_manager/cmd/data_plane/"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/data_plane/config_cluster$2.yaml /cluster_manager/cmd/data_plane/config_cluster.yaml"

    # Remove old logs
    RemoteExec $1 "sudo journalctl --vacuum-time=1s && sudo journalctl --vacuum-time=1d"
    # Update systemd
    RemoteExec $1 "sudo cp -a ~/cluster_manager/sripts/systemd/* /etc/systemd/system/"
    # Start data plane
    RemoteExec $1 "sudo systemctl daemon-reload && sudo systemctl restart data_plane.service"
}

function SetupWorkerNodes() {
    function internal_setup() {
        # LFS pull for VM kernel image and rootfs
        RemoteExec $1 "cd ~/cluster_manager; git pull; git lfs pull"

        # Compile worker node daemon
        RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/worker_node"
        RemoteExec $1 "cd ~/cluster_manager/cmd/worker_node/; /usr/local/go/bin/go build main.go"
        RemoteExec $1 "sudo cp ~/cluster_manager/cmd/worker_node/main /cluster_manager/cmd/worker_node/"
        RemoteExec $1 "sudo cp ~/cluster_manager/cmd/worker_node/config_cluster$2.yaml /cluster_manager/cmd/worker_node/config_cluster.yaml"

        # For readiness probe
        RemoteExec $1 "sudo sysctl -w net.ipv4.conf.all.route_localnet=1"
        # For reachability of sandboxes from other cluster nodes
        RemoteExec $1 "sudo sysctl -w net.ipv4.ip_forward=1"

        # Remove old snapshots
        RemoteExec $1 "sudo rm -rf /tmp/snapshots"

        # Remove old logs
        RemoteExec $1 "sudo journalctl --vacuum-time=1s && sudo journalctl --vacuum-time=1d"
        # Update systemd
        RemoteExec $1 "sudo cp -a ~/cluster_manager/scripts/systemd/* /etc/systemd/system/"
        # Start worker node daemon
        RemoteExec $1 "sudo systemctl daemon-reload && sudo systemctl restart worker_node.service"
    }

    CP_PREFIX=""
    if [ "$1" -ne 1 ]; then
        CP_PREFIX="_raft"
    fi
    shift

    for NODE in "$@"
    do
        internal_setup $NODE $CP_PREFIX &
    done

    wait
}

function KillSystemdServices() {
    function internal_kill() {
        RemoteExec $1 "sudo systemctl stop control_plane data_plane worker_node haproxy && sudo killall firecracker"
    }

    for NODE in "$@"
    do
        internal_kill $NODE &
    done

    wait
}

function StoreResults() {
    if [ "$HA" = true ] ;
    then
      scp $DATAPLANE_1:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv plotting/proxy_trace_$1_1.csv
      scp $DATAPLANE_2:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv plotting/proxy_trace_$1_2.csv
      scp $DATAPLANE_3:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv plotting/proxy_trace_$1_3.csv

      scp $CONTROLPLANE_1:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv plotting/cold_start_trace_$1_1.csv
      scp $CONTROLPLANE_2:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv plotting/cold_start_trace_$1_2.csv
      scp $CONTROLPLANE_3:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv plotting/cold_start_trace_$1_3.csv
    else
      scp $DATAPLANE:~/cluster_manager/cmd/data_plane/data/proxy_trace.csv plotting/proxy_trace_$1_1.csv
      scp $CONTROLPLANE:~/cluster_manager/cmd/master_node/data/cold_start_trace.csv plotting/cold_start_trace_$1_1.csv
    fi
}