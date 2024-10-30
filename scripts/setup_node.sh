#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"
source "$DIR/setup.cfg"

function SetupLoadBalancer() {
    sudo apt-get -y install keepalived haproxy >> /dev/null

    CONFIGS_PATH=$DIR/../configs

    bash $CONFIGS_PATH/substitute_interface.sh

    sudo cp $CONFIGS_PATH/check_apiserver.sh /etc/keepalived/check_apiserver.sh
    sudo cp $CONFIGS_PATH/keepalived.conff /etc/keepalived/keepalived.conf
    sudo cp $CONFIGS_PATH/haproxy.cfg /etc/haproxy/haproxy.cfg
    sudo systemctl daemon-reload

    sudo systemctl restart keepalived
}

sudo apt-get update >> /dev/null
sudo apt-get install git-lfs htop
sudo apt-get install -y python3-pip && pip3 install psutil

# Install Golang
if [ -x "$(command -v go)" ]; then
    echo "Go has already been installed"
else
    wget --continue --quiet https://go.dev/dl/go1.22.2.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.22.2.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    sudo sh -c  "echo 'export PATH=\$PATH:/usr/local/go/bin' >> /etc/profile"
fi

# Install Docker
sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
sudo apt-get install -y docker.io
sudo groupadd -f docker
sudo usermod -aG docker $USER

# Install CNI
sudo apt-get install -y ca-certificates curl >> /dev/null
sudo mkdir -p -m 755 /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | sudo gpg --batch --yes --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update >> /dev/null
sudo apt-get -y install containerd kubernetes-cni >> /dev/null

# Install yq for updating worker node configuration
sudo wget -qO /usr/local/bin/yq https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64
sudo chmod a+x /usr/local/bin/yq

if [ "$RUNTIME" = "firecracker" ]
then
    # Install Firecracker
    ARCH="$(uname -m)"
    release_url="https://github.com/firecracker-microvm/firecracker/releases"
    if [ "$FIRECRACKER_VERSION" = "" ]; then
        latest=$(basename $(curl -fsSLI -o /dev/null -w  %{url_effective} ${release_url}/latest))
    else
        latest=$FIRECRACKER_VERSION
    fi

    curl -L ${release_url}/download/${latest}/firecracker-${latest}-${ARCH}.tgz | tar -xz
    sudo rm -rf /usr/local/bin/firecracker
    sudo mv release-${latest}-$(uname -m) /usr/local/bin/firecracker
    sudo mv /usr/local/bin/firecracker/firecracker-${latest}-${ARCH} /usr/local/bin/firecracker/firecracker
    echo "export PATH=$PATH:/usr/local/bin/firecracker" | sudo tee -a /etc/profile
elif [ "$RUNTIME" = "fcctr" ]
then
    # Set up a devmapper pool for firecracker-containerd
    DEVMAPPER_DIR=/var/lib/firecracker-containerd/snapshotter/devmapper
    sudo dmsetup remove_all
    sudo rm -rf /var/lib/firecracker-containerd
    sudo mkdir -p "$DEVMAPPER_DIR"
    sudo touch "$DEVMAPPER_DIR/data"
    sudo truncate -s 100G "$DEVMAPPER_DIR/data"
    sudo touch "$DEVMAPPER_DIR/metadata"
    sudo truncate -s 2G "$DEVMAPPER_DIR/metadata"
    DATADEV="$(sudo losetup --find --show $DEVMAPPER_DIR/data)"
    METADEV="$(sudo losetup --find --show $DEVMAPPER_DIR/metadata)"
    SECTORSIZE=512
    DATASIZE="$(sudo blockdev --getsize64 -q $DATADEV)"
    LENGTH_SECTORS=$(($DATASIZE / $SECTORSIZE))
    sudo dmsetup create fc-dev-thinpool \
        --table "0 $LENGTH_SECTORS thin-pool $METADEV $DATADEV 128 32768 1 skip_block_zeroing"
    # Install firecracker-containerd
    [ ! -d ~/vHive ] && git clone https://github.com/vhive-serverless/vHive.git ~/vHive
    cd ~/vHive
    git checkout 7b63f06
    ~/vHive/scripts/setup_firecracker_containerd.sh
elif [ "$RUNTIME" = "dandelion" ]
then
    # Install Cargo for Dandelion setup
    sudo apt install cargo -y
fi

# Copy systemd services
sudo cp -a ~/cluster_manager/scripts/systemd/* /etc/systemd/system/

# For local readiness probes
sudo sysctl -w net.ipv4.conf.all.route_localnet=1
# For reachability of sandboxes from other cluster nodes
sudo sysctl -w net.ipv4.ip_forward=1

readonly NODE_PURPOSE=$1
if [ "$NODE_PURPOSE" = "CONTROL_PLANE" ]; then
    SetupLoadBalancer
fi

if [ "$NODE_PURPOSE" = "INVITRO" ]; then
    [ ! -d ~/invitro ] && git clone https://github.com/vhive-serverless/invitro.git ~/invitro
fi
