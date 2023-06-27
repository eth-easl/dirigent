# Cluster manager

## How to run

### Installation

kubernetes-cni must be installed.

```bash
curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.1/cni-plugins-linux-amd64-v0.8.1.tgz
sudo mkdir -p /opt/cni/bin
sudo tar -C /opt/cni/bin -xzf cni-plugins.tgz
```

If you want to install it on a custom path.

```bash
INSTALL_PATH='your/path/here'

curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.1/cni-plugins-linux-amd64-v0.8.1.tgz
sudo mkdir -p /opt/cni/bin
sudo tar -C INSTALL_PATH -xzf cni-plugins.tgz
```

### Run the code

TODO @Francois Costa: Fill this part in the future 

## Linter

```bash
golangci-lint run --fix
```

``` bash
golangci-lint run -v --timeout 5m0s
```