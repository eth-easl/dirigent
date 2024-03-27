#!/bin/sh
set -e

# Deploys a Docker registry to the first node specified in the arguments, and
# changes the Docker configuration on all other specified nodes to allow the
# first node as a plain HTTP registry.

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" > /dev/null 2>&1 && pwd)"

source "$DIR/common.sh"
source "$DIR/setup.cfg"

# TODO: Make this smarter
REGISTRY_ADDR=$(RemoteExec $1 "ip addr show | grep 'inet 10[.]0[.]' | tr -s ' ' | cut -d' ' -f3 | cut -d/ -f1")
SETUP_DOCKER_CMD="
set -e

# Install Docker
sudo apt-get update
sudo apt-get install -y docker.io
sudo groupadd -f docker
sudo usermod -aG docker \$USER

# Setup registry
json='{\"insecure-registries\": [\"$REGISTRY_ADDR\"]}'
echo "\$json" | sudo tee /etc/docker/daemon.json
sudo systemctl restart docker
"

RemoteExec $1 "$SETUP_DOCKER_CMD" > /dev/null

# Run Docker registry
RemoteExec $1 "
set -e
sudo docker container stop registry || true
sudo docker container rm -v registry || true
sudo docker run -d -p 80:5000 --restart always --name registry registry:2
"

while [ "$#" -gt 1 ]
do
    shift
    RemoteExec $1 "$SETUP_DOCKER_CMD" > /dev/null &
done
wait
