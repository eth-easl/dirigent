#!/bin/bash

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

source $DIR/common.sh
source $DIR/setup.cfg

REMOTE_SSH_KEY="~/.ssh/id_ed25519"
LOCAL_SSH_KEY="$DIR/ssh-dirigent"

function GenerateSSHKeys() {
    if [ ! -f "$LOCAL_SSH_KEY" ]
    then
        ssh-keygen -t ed25519 -f "$LOCAL_SSH_KEY" -N ''
        echo -e "${color_red}ACTION REQUIRED:${color_white} Please add a new SSH key to your GitHub profile."
        echo -e "${color_green}Step 1:${color_reset} Visit ${color_cyan}https://github.com/settings/ssh/new${color_reset}"
        echo -e "${color_green}Step 2:${color_reset} Paste the following contents:"
        cat "$LOCAL_SSH_KEY.pub"
        read -p "Press Enter after making these changes."
    fi
}

function CloneDirigent() {
    if RemoteExec $1 "[ ! -d ~/cluster_manager ]"
    then
        # Copy the private SSH key for cloning Dirigent.
        if RemoteExec $1 "[ ! -f '$REMOTE_SSH_KEY' ]"
        then
            RemoteExec $1 'ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts'
            CopyToRemote "$LOCAL_SSH_KEY" "$1:$REMOTE_SSH_KEY"
        fi

        # Clone the current remote tracking branch of Dirigent if there is one,
        # or default branch if there is not.
        current_branch=$(git rev-parse --abbrev-ref HEAD)
        default_branch=$(git symbolic-ref refs/remotes/origin/HEAD | sed 's@^refs/remotes/origin/@@')
        remote_branch=$(git rev-parse --abbrev-ref --symbolic-full-name @{u} >/dev/null 2>&1 && echo "$current_branch" || echo "$default_branch")
        RemoteExec $1 "git clone --branch=$remote_branch git@github.com:eth-easl/dirigent.git ~/cluster_manager"
    fi
}

function SetupNode() {
    CloneDirigent $1
    CopyToRemote "$DIR/setup_node.sh" "$1:~/cluster_manager/scripts/setup_node.sh"
    strict_mode=$([[ "$-" == *e* ]] && echo "bash -e" || echo "bash")
    RemoteExec $1 "$strict_mode ~/cluster_manager/scripts/setup_node.sh $2"
    # LFS pull for VM kernel image and rootfs
    RemoteExec $1 'cd ~/cluster_manager; git pull; git lfs pull'
    RemoteExec $1 'sudo cp -r ~/cluster_manager/ /cluster_manager'
    RemoteExec $1 '[ ! -d ~/invitro ] && git clone https://github.com/vhive-serverless/invitro.git'
    CopyToRemote "$DIR/invitro_traces" "$1:invitro"
}

git lfs pull
python3 "$DIR/invitro_traces/generate_traces.py"

GenerateSSHKeys

NODE_COUNTER=0

for NODE in "$@"
do
    if [ "$NODE_COUNTER" -eq 0 ]; then
        HA_SETTING="REDIS"
    elif [ "$NODE_COUNTER" -le $CONTROL_PLANE_REPLICAS ]; then
        HA_SETTING="CONTROL_PLANE"
    elif [ "$NODE_COUNTER" -le $(( $CONTROL_PLANE_REPLICAS + $DATA_PLANE_REPLICAS )) ]; then
        HA_SETTING="DATA_PLANE"
    else
        HA_SETTING="WORKER_NODE"
    fi

    SetupNode $NODE $HA_SETTING &
    let NODE_COUNTER++ || true
done

wait
