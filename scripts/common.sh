#!/bin/bash

function RemoteExec() {
    ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

function SetupRedis() {
    # Start Redis server
    RemoteExec $1 "docker stop \$(docker ps -aq)"
    RemoteExec $1 "docker rm \$(docker ps -a -q)"
    RemoteExec $1 "docker run -d --name redis-stack-server -p 6379:6379 redis/redis-stack-server:latest"
}

function SetupControlPlane() {
    RemoteExec $1 "cd ~/cluster_manager; git pull"

    # Compile control plane
    RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/master_node"
    RemoteExec $1 "cd ~/cluster_manager/cmd/master_node/; /usr/local/go/bin/go build main.go"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/master_node/main /cluster_manager/cmd/master_node/"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/master_node/config_cluster.yaml /cluster_manager/cmd/master_node/"

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
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/data_plane/config_cluster.yaml /cluster_manager/cmd/data_plane/"

    # Remove old logs
    RemoteExec $1 "sudo journalctl --vacuum-time=1s && sudo journalctl --vacuum-time=1d"
    # Update systemd
    RemoteExec $1 "sudo cp -a ~/cluster_manager/scripts/systemd/* /etc/systemd/system/"
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
        RemoteExec $1 "sudo cp -r ~/cluster_manager/* /cluster_manager"

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

    for NODE in "$@"
    do
        internal_setup $NODE &
    done

    wait
}

function KillSystemdServices() {
    function internal_kill() {
        RemoteExec $1 "sudo killall firecracker && sudo systemctl stop worker_node"
    }

    RemoteExec $1 "sudo systemctl stop control_plane"
    shift
    RemoteExec $1 "sudo systemctl stop data_plane"
    shift

    for NODE in "$@"
    do
        internal_kill $NODE &
    done

    wait
}
