#!/bin/bash

readonly DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

function SetupLoadBalancer() {
    sudo apt-get update >> /dev/null
    sudo apt-get -y install keepalived haproxy >> /dev/null

    CONFIGS_PATH=$DIR/../configs

    bash $CONFIGS_PATH/substitute_interface.sh

    sudo cp $CONFIGS_PATH/check_apiserver.sh /etc/keepalived/check_apiserver.sh
    sudo cp $CONFIGS_PATH/keepalived.conff /etc/keepalived/keepalived.conf
    sudo cp $CONFIGS_PATH/haproxy.cfg /etc/haproxy/haproxy.cfg
    sudo systemctl daemon-reload

    sudo systemctl restart keepalived
}

sudo apt-get update
sudo apt-get install git-lfs htop

# Install Golang
if [ -x "$(command -v go)" ]; then
    echo "Go has already been installed"
else
    wget --continue --quiet https://go.dev/dl/go1.20.4.linux-amd64.tar.gz
    sudo tar -C /usr/local -xzf go1.20.4.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    sudo sh -c  "echo 'export PATH=\$PATH:/usr/local/go/bin' >> /etc/profile"
fi

# Install Docker
sudo curl -L "https://github.com/docker/compose/releases/download/1.29.2/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
sudo apt-get install -y docker.io
sudo groupadd docker
sudo usermod -aG docker $USER
newgrp docker

# Install CNI
sudo apt-get update >> /dev/null
sudo apt-get install -y apt-transport-https ca-certificates curl >> /dev/null
sudo mkdir -p -m 755 /etc/apt/keyrings
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.29/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.29/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update >> /dev/null
sudo apt-get -y install containerd kubernetes-cni >> /dev/null

# Install Firecracker
ARCH="$(uname -m)"
release_url="https://github.com/firecracker-microvm/firecracker/releases"
latest=$(basename $(curl -fsSLI -o /dev/null -w  %{url_effective} ${release_url}/latest))
curl -L ${release_url}/download/${latest}/firecracker-${latest}-${ARCH}.tgz \
| tar -xz
sudo mv release-${latest}-$(uname -m) /usr/local/bin/firecracker
sudo mv /usr/local/bin/firecracker/firecracker-${latest}-${ARCH} /usr/local/bin/firecracker/firecracker
sudo sh -c  "echo 'export PATH=\$PATH:/usr/local/bin/firecracker' >> /etc/profile"

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
