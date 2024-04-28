<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/othneildrew/Best-README-Template">
    <img src="https://systems.ethz.ch/_jcr_content/orgbox/image.imageformat.logo.1825449303.svg" alt="Logo" width="160" height="80">
  </a>

<h1 align="center">Dirigent</h1>

---

[![GitHub Workflow Status](https://github.com/eth-easl/modyn/actions/workflows/workflow.yaml/badge.svg)](https://github.com/eth-easl/modyn/actions/workflows/workflow.yaml)
[![License](https://img.shields.io/github/license/eth-easl/modyn)](https://img.shields.io/github/license/eth-easl/modyn)

  <p align="center">
    A research cluster manager built at ETH Zürich
    <br />
    <a href="https://systems.ethz.ch/"><strong>Systems groups page »</strong></a>
    <br />
    <br />
    <a href="">View Demo</a>
    ·
    <a href="">Report Bug</a>
  </p>

  <p>Serverless computing optimizes cloud resource use for better performance. Yet, current serverless cluster managers are retrofitted from old systems, not built for serverless tasks. We examine Knative-on-K8s, a modern serverless cluster manager. It causes delays and 65%+ of latency for cold start function calls. These issues arise during high sandbox changes, common in production serverless setups. We identify the problem and suggest new design principles to enhance performance by rethinking cluster manager architecture.</p>
</div>


<!-- ABOUT THE PROJECT -->
## About The Project

The cluster manager in question has been developed within the systems group at ETH Zürich. It has been designed and fine-tuned specifically to the requirements of Function as a Service (FaaS) paradigms.

### Built With

<div align="center">

![Go](https://img.shields.io/badge/go-%2300ADD8.svg?style=for-the-badge&logo=go&logoColor=white)
![Docker](https://img.shields.io/badge/docker-%230db7ed.svg?style=for-the-badge&logo=docker&logoColor=white)

</div>

## Software implementation

See the `README.md` to get started with the code. 

The folder structure is as follows:

* `cmd` is the list of programs you can start
* `proto` represents the proto handlers
* `internal/master_node` corresponds to the source code of the master node
* `internal/data_plane` corresponds to the source code of the data plane
* `internal/woker_node` corresponds to the source code of the worker node
* `pkg` are shared packages that are used inside of internal to perform multiple actions
* `scripts` are a list of scripts you can use to measures or tests Dirigent

## Getting the code

You can download a copy of all the files in this repository by cloning the
[git](https://github.com/eth-easl/cluster_manager) repository:
```bash
    git clone https://github.com/eth-easl/cluster_manager
```

## Dependencies - Installation


Install HAProxy

```bash
sudo apt update && sudo apt install -y haproxy
sudo cp configs/haproxy.cfg /etc/haproxy/haproxy.cfg

```

Install kubernetes-cni.

```bash
curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.1/cni-plugins-linux-amd64-v0.8.1.tgz
sudo mkdir -p /opt/cni/bin
sudo tar -C /opt/cni/bin -xzf cni-plugins.tgz
```

Custom path installation: 

```bash
INSTALL_PATH='your/path/here'

curl -L -o cni-plugins.tgz https://github.com/containernetworking/plugins/releases/download/v0.8.1/cni-plugins-linux-amd64-v0.8.1.tgz
sudo mkdir -p /opt/cni/bin
sudo tar -C INSTALL_PATH -xzf cni-plugins.tgz
```

## Configure Firecracker for local development

- Install Firecracker
```bash
ARCH="$(uname -m)"
release_url="https://github.com/firecracker-microvm/firecracker/releases"
latest=$(basename $(curl -fsSLI -o /dev/null -w  %{url_effective} ${release_url}/latest))
curl -L ${release_url}/download/${latest}/firecracker-${latest}-${ARCH}.tgz \
| tar -xz
sudo mv release-${latest}-$(uname -m) /usr/local/bin/firecracker
sudo mv /usr/local/bin/firecracker/firecracker-${latest}-${ARCH} /usr/local/bin/firecracker/firecracker
sudo sh -c  "echo 'export PATH=\$PATH:/usr/local/bin/firecracker' >> /etc/profile" 
```
- Install tun-tap
```bash
git clone https://github.com/awslabs/tc-redirect-tap.git || true
make -C tc-redirect-tap
sudo cp tc-redirect-tap/tc-redirect-tap /opt/cni/bin
```
- Install ARP
```bash
sudo apt-get update && sudo apt-get install net-tools
```
- Download Kernel
```bash
sudo apt-get update && sudo apt-get install git-lfs
git lfs fetch
git lfs checkout
git lfs pull 
```
- Run control plane and data plane processes. Run worker daemon with `sudo` and by hardcoding environmental variable `PATH` to point to the directory where Firecracker is located.
```bash
sudo env 'PATH=\$PATH:/usr/local/bin/firecracker' /usr/local/go/bin/go run cmd/worker_node/main.go 
```

## System configuration

To run the cluster manager locally the following setting must be enabled:
```bash
sudo sysctl -w net.ipv4.conf.all.route_localnet=1
```

## Clone the code

Firstly, you need to install the control plane and Redis on one machine, data plane on another machine and workers on several other machines. You can simply call the script `remote_install.sh` with the addresses of the machines:

```bash
./remote_install.sh machine1 machine2 ...
```

If your local username differs from your remote username on these machines, you also need to include your remote username in the address:

```bash
./remote_install.sh username@machine1 username@machine2 ...
```

## Launch code

> launch db

```bash
sudo docker-compose up
```

> launch master node

```bash
sudo go run cmd/master_node/main.go 
```

> launch data plane

```bash
go run cmd/data_plane/main.go 
```

> launch master node

```bash
sudo go run cmd/worker_node/main.go
```

### Fire invocation

This command will fire a single invocation.

```bash
cd clients/sync
go run main.go
```

In case you get a timeout, try to run the following command before

```bash
# For local readiness probes
sudo sysctl -w net.ipv4.conf.all.route_localnet=1
# For reachability of sandboxes from other cluster nodes
sudo sysctl -w net.ipv4.ip_forward=1
```

## If the network breaks locally

```bash
sudo iptables -t nat -F
```

## License

Distributed under the MIT License. See `LICENSE.txt` for more information.

## Contact

Lazar Cvetković - lazar.cvetkovic@inf.ethz.ch

François Costa - fcosta@ethz.ch

Ana Klimovic - aklimovic@ethz.ch

## For developpers

### Generate proto files

First you have to install the protobuf compiler

```bash
make install_golang_proto_compiler
```

Then you can compile the proto types using the following command

```bash
make proto
```

### Generate mock tests

First you have to install the mockgen library

```bash
make install_mockgen
```

Then you can create the files with the following command

```bash
make generate_mock_files
```

### Run the tests

```bash
sudo go test -v ./...
```
