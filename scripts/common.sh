#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source $DIR/setup.cfg

readonly INVITRO=Francois@hp157.utah.cloudlab.us
readonly CONTROLPLANE=Francois@hp124.utah.cloudlab.us
readonly DATAPLANE=Francois@hp119.utah.cloudlab.us

readonly CONTROLPLANE_1=Francois@hp091.utah.cloudlab.us
readonly CONTROLPLANE_2=Francois@hp081.utah.cloudlab.us
readonly CONTROLPLANE_3=Francois@hp023.utah.cloudlab.us

readonly DATAPLANE_1=Francois@hp149.utah.cloudlab.us
readonly DATAPLANE_2=Francois@hp077.utah.cloudlab.us
readonly DATAPLANE_3=Francois@hp134.utah.cloudlab.us

readonly HA=false

# Colors as useful indicators to users running these scripts.
readonly color_white="\033[0;37m"
readonly color_green="\033[1;32m"
readonly color_red="\033[1;31m"
readonly color_cyan="\033[1;36m"
readonly color_reset="\033[0m"

function RemoteExec() {
    ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2"
}

function CopyToRemote() {
    rsync -av "$1" "$2"
}

function SetupControlPlane() {
    # Start Redis server
    RemoteExec $1 "sudo docker stop \$(sudo docker ps -aq)" || true
    RemoteExec $1 "sudo docker rm \$(sudo docker ps -a -q)" || true
    RemoteExec $1 "sudo docker run -d --name redis-stack-server -p 6379:6379 redis/redis-stack-server:latest"

    RemoteExec $1 "cd ~/cluster_manager; git pull"

    # Compile control plane
    RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/master_node"
    RemoteExec $1 "cd ~/cluster_manager/cmd/master_node/; /usr/local/go/bin/go build main.go"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/master_node/main /cluster_manager/cmd/master_node/"
    RemoteExec $1 "sudo cp ~/cluster_manager/cmd/master_node/config_cluster$2.yaml /cluster_manager/cmd/master_node/config_cluster.yaml"

    # Remove old logs
    RemoteExec $1 "sudo journalctl --rotate && sudo journalctl --vacuum-time=1s && sudo journalctl --vacuum-time=1d"
    # Update systemd
    RemoteExec $1 "sudo cp -a ~/cluster_manager/scripts/systemd/* /etc/systemd/system/"
    # Increase TCP port pool size
    RemoteExec $1 "sudo sysctl -w net.ipv4.ip_local_port_range='1024 65535'"
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
    RemoteExec $1 "sudo cp -a ~/cluster_manager/scripts/systemd/* /etc/systemd/system/"
    # Increase TCP port pool size
    RemoteExec $1 "sudo sysctl -w net.ipv4.ip_local_port_range='1024 65535'"
    # Start data plane
    RemoteExec $1 "sudo systemctl daemon-reload && sudo systemctl restart data_plane.service"
}

function SetupWorkerNodes() {
    function internal_setup() {
        # LFS pull for VM kernel image and rootfs
        RemoteExec $1 "cd ~/cluster_manager; git pull; git lfs pull"
        RemoteExec $1 "cd ~/dandelion; git pull; git lfs pull"

        # Compile worker node daemon
        RemoteExec $1 "sudo mkdir -p /cluster_manager/cmd/worker_node"
        RemoteExec $1 "cd ~/cluster_manager/cmd/worker_node/; /usr/local/go/bin/go build main.go"
        RemoteExec $1 "sudo cp ~/cluster_manager/cmd/worker_node/main /cluster_manager/cmd/worker_node/"
        RemoteExec $1 "sudo cp ~/cluster_manager/cmd/worker_node/config_cluster$2.yaml /cluster_manager/cmd/worker_node/config_cluster.yaml"

        # Start Dandelion
        #RUNTIME=$(RemoteExec $1 "cat /cluster_manager/cmd/worker_node/config_cluster.yaml | yq '.criType'")

        #if [[ "$RUNTIME" == "dandelion" ]]; then
        RemoteExec $1 "cd ~/dandelion; RUSTFLAGS='-C target-feature=+crt-static' cargo build --bin mmu_worker --features mmu --target \$(arch)-unknown-linux-gnu"
        RemoteExec $1 "tmux new -s dandelion -d"
        RemoteExec $1 "tmux send-keys -t dandelion 'cd ~/dandelion; RUST_LOG=debug cargo run --bin dandelion_server -F mmu,reqwest_io' ENTER"
        #fi

        # For readiness probe
        RemoteExec $1 "sudo sysctl -w net.ipv4.conf.all.route_localnet=1"
        # For reachability of sandboxes from other cluster nodes
        RemoteExec $1 "sudo sysctl -w net.ipv4.ip_forward=1"

        # Remove old snapshots
        RemoteExec $1 "sudo rm -rf /tmp/snapshots"
        # Remove old containers and network
        RemoteExec $1 "source ~/cluster_manager/scripts/common.sh; WipeContainerdCNI"

        # Remove old logs
        RemoteExec $1 "sudo journalctl --vacuum-time=1s && sudo journalctl --vacuum-time=1d"
        # Increase TCP port pool size
        RemoteExec $1 "sudo sysctl -w net.ipv4.ip_local_port_range='1024 65535'"
        # Update systemd
        RemoteExec $1 "sudo cp -a ~/cluster_manager/scripts/systemd/* /etc/systemd/system/"
        # Start worker node daemon
        RemoteExec $1 "sudo systemctl daemon-reload && sudo systemctl restart worker_node.service"
    }

    CP_PREFIX=""
    if [ "$1" -ne 1 ]; then
        CP_PREFIX="_raft"
    fi
    shift # throw control plane replicas argument

    for NODE in "$@"
    do
        internal_setup $NODE $CP_PREFIX &
    done

    wait
}

function KillSystemdServices() {
    function internal_kill() {
        RemoteExec $1 "sudo systemctl stop control_plane data_plane worker_node haproxy && sudo killall firecracker"

        RemoteExec $1 "sudo pkill -9 dandelion_server"
        RemoteExec $1 "tmux kill-session -t dandelion"
    }

    for NODE in "$@"
    do
        internal_kill $NODE &
    done

    wait
}

function WipeContainerdCNI() {
    # Remove all containers
    sudo pkill -9 workload || true
    ctr --namespace cm container ls -q | xargs ctr --namespace cm container delete || true

    # Remove all unused images
    ctr --namespace cm image prune --all

    # Remove all CNI networks
    sudo iptables -t nat -F
    for chain in $(sudo iptables -t nat -L | grep "CNI-" | cut -d' ' -f2)
    do
        sudo iptables -t nat -X $chain
    done
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

function SetupFakeWorkerNodes() {
    function internal_setup() {
        RemoteExec $1 "cd ~/cluster_manager; git pull; git lfs pull"

        for INDEX in `seq 0 $(($DAEMONS_PER_NODE - 1))`;
        do
            local DAEMON_PORT=$((10010 + $INDEX))

            RemoteExec $1 "export DAEMON_PORT=${DAEMON_PORT}; cd cluster_manager; cat cmd/worker_node/config_cluster_fake_worker$2.yaml | envsubst > cmd/worker_node/tmp && mv cmd/worker_node/tmp cmd/worker_node/config_cluster_fake_worker_${INDEX}.yaml"

            local CPU_CORE=$((INDEX % 20))
            local ARGS="--config cmd/worker_node/config_cluster_fake_worker_${INDEX}.yaml"
            local CMD="cd ~/cluster_manager; sudo env 'PATH=\$PATH:/usr/local/bin/firecracker:/usr/bin' taskset -c ${CPU_CORE} /usr/local/go/bin/go run cmd/worker_node/main.go ${ARGS}"

            RemoteExec $1 "tmux new -s worker_daemon_${INDEX} -d"
            RemoteExec $1 "tmux send -t worker_daemon_${INDEX} \"$CMD\" ENTER"
        done

        readonly WORKLOAD_COMMAND="cd ~/cluster_manager/workload; sudo /usr/local/go/bin/go run workload.go http_workload.go"

        RemoteExec $1 "tmux new -s workload -d"
        RemoteExec $1 "tmux send -t workload \"$WORKLOAD_COMMAND\" ENTER"
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

function KillFakeWorkerNodes() {
    function internal_kill() {
        local PID_TO_KILL=$(RemoteExec $1 "ps -aux | grep cmd/worker_node | awk '{print \$2}' | tr '\n' ' '")
        for INDEX in `seq 0 $(($DAEMONS_PER_NODE - 1))`;
        do
            RemoteExec $1 "sudo kill -9 ${PID_TO_KILL}"
            RemoteExec $1 "tmux kill-session -t worker_daemon_${INDEX}"
        done

        local WORKLOAD_TO_KILL=$(RemoteExec $1 "ps -aux | grep workload | awk '{print \$2}' | tr '\n' ' '")
        RemoteExec $1 "sudo kill -9 ${WORKLOAD_TO_KILL}"
        RemoteExec $1 "tmux kill-session -t workload"
    }

    for NODE in "$@"
    do
        internal_kill $NODE &
    done

    wait
}
