#!/bin/bash

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
sudo apt-get install -y docker.io git-lfs

# Install CNI
K8S_VERSION=1.23.5-00
curl --silent --show-error https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add -
sudo sh -c "echo 'deb http://apt.kubernetes.io/ kubernetes-xenial main' > /etc/apt/sources.list.d/kubernetes.list"
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
