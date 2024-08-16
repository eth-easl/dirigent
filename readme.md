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

The folder structure is as follow:

* `cmd` is the list of programms you can start
* `api` represents the API handlers
* `internal/master_node` corresponds to the source code of the master node
* `internal/data_plane` corresponds to the source code of the data plane
* `internal/woker_node` corresponds to the source code of the worker node
* `pkg` are shared packages that are used inside of internal to perform multiple actions
* `scripts` are a list of scripts you can use to measures or tests the cluster manager

## Getting the code

You can download a copy of all the files in this repository by cloning the
[git](https://github.com/eth-easl/dirigent) repository:
```bash
    git clone https://github.com/eth-easl/dirigent
```

## Dependencies - Installation

To run the cluster manager locally the following setting must be enabled:
```bash
sudo sysctl -w net.ipv4.conf.all.route_localnet=1
```

Install HAProxy

```bash
sudo apt update && sudo apt install -y haproxy
sudo cp configs/haproxy.cfg /etc/haproxy/haproxy.cfg

```

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

## Start the code

To launch the code, perform the following actions. 


Once the configuration stage is complete, we can start the programs.

#### Clone the code

First we need to install a copy of a master node on one machine, a dataplane on another machine, a copy of redis on a third machine and finally copy the code base on several other machines for the workers. You can simply call the script remote_install.sh with the ssh address of the computers. Before calling the script you have to make sure you have a github token on the following path which can install ssh keys.

```bash
ACCESS_TOKEN="$(cat ~/.git_token_loader)"
```

```bash
./remote_install.sh ip1 ip2 ...
```

#### Setup configuration file

Once this has been done, we can move on to configuring the various programs. The most important field is to set the correct IP for the database and control plane.

Config master node 

```yaml
port: "9090" # Port used for the GRPC server
portRegistration: "9091" # Port for registrating a new service
verbosity: "trace" # Verbosity of the logs
traceOutputFolder: "data" # Output folder for measurements
placementPolicy: "kubernetes" # Placement policy
persistence: true # Store persistence value - if the value is false you can run the cluster without database
reconstruct: false # Reconstruct values on start

profiler:
  enable: true # Enable profiler support - it makes the programm a bit slower
  mutex: false # Enable mutex support in profiler

redis:
  address: "127.0.0.1:6379" # Address of the database 
  password: "" # Password
  db: 0 # Database name

```

Config dataplane

```yaml
controlPlaneIp: "localhost" # Ip of the control plane (master node)
controlPlanePort: "9090" # GRPC port used in the control plane
portProxy: "8080" # Port used for requests 
portGRPC: "8081" # Port used for the GRPC server
verbosity: "trace" # Verbosity of the logs
traceOutputFolder: "data" # Output folder for measurements
loadBalancingPolicy: "knative" # Load balancing policy
```

Config worker node

```yaml
controlPlaneIp: "localhost" # Ip of the control plane (master node)
controlPlanePort: "9090" # GRPC port used in the control plane
port: "10010" # Port used for the GRPC server
verbosity: "trace" # Verbosity of the logs
criPath: "/run/containerd/containerd.sock" # path for CRI
cniConfigPath: "configs/cni.conf" # path for CNI
prefetchImage: true # If enabled, workers will prefetch an image (thus image download will be removed from the measures)
```

#### Launch code

> launch db

```bash
sudo docker-compose up
```

> launch master node

```bash
cd cmd/master_node; go run main.go --config config_cluster.yaml
```

> launch data plane

```bash
cd cmd/data_plane; go run main.go --config config_cluster.yaml
```

> launch master node

```bash
cd scripts/francois; ./restart_workers.sh ip1 ip2 ip3 ....
```

#### Fire invocation

This command will fire a single invocation.

```bash
cd scripts/francois; ./burst.sh 1 
```

In case you get a timeout, try to run the following command before

```bash
# For local readiness probes
sudo sysctl -w net.ipv4.conf.all.route_localnet=1
# For reachability of sandboxes from other cluster nodes
sudo sysctl -w net.ipv4.ip_forward=1
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

### Linter

```bash
golangci-lint run --fix
```

or with verbose

``` bash
golangci-lint run -v --timeout 5m0s
```

### Profiler

Nice tutorial that explains how to use it

> https://teivah.medium.com/profiling-and-execution-tracing-in-go-a5e646970f5b
