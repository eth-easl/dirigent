function RemoteExec() {
  ssh -oStrictHostKeyChecking=no -p 22 "$1" "$2";
}

function RestartWorkers() {
    function internal_setup() {
        RemoteExec $1 "tmux kill-session -t worker"
        RemoteExec $1 "tmux new -s worker -d"

        CMD=$"cd ~/cluster_manager; git fetch origin;git reset --hard origin/sweep_test_10; cd ~/cluster_manager/cmd/worker_node; sudo /usr/local/go/bin/go run main.go --configPath config_cluster.yaml"

        RemoteExec $1 "tmux send -t worker \"$CMD\" ENTER"
    }

    for NODE in "$@"
    do
        internal_setup $NODE &
    done

    wait
}

function StopWorkers() {
    function internal_setup() {
        RemoteExec $1 "tmux kill-session -t worker"
    }

    for NODE in "$@"
    do
        internal_setup $NODE &
    done

    wait
}


#StopWorkers $@
RestartWorkers $@