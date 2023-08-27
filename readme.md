<!-- PROJECT LOGO -->
<br />
<div align="center">
  <a href="https://github.com/othneildrew/Best-README-Template">
    <img src="https://systems.ethz.ch/_jcr_content/orgbox/image.imageformat.logo.1825449303.svg" alt="Logo" width="160" height="80">
  </a>

<h3 align="center">Cluster Manager</h3>

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
[git](https://github.com/eth-easl/cluster_manager) repository:
```bash
    git clone https://github.com/eth-easl/cluster_manager
```

## Dependencies - Installation

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

First we need to install a copy of a master node on one machine, a dataplane on another machine, a copy of redis on a third machine and finally copy the code base on several other machines for the workers.

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
cd cmd/master_node; go run main.go --configPath config_cluster.yaml
```

> launch data plane

```bash
cd cmd/data_plane; go run main.go --configPath config_cluster.yaml
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

## License

Distributed under the MIT License. See `LICENSE.txt` for more information.

## Contact

Lazar Cvetkovic - lazar.cvetkovic@inf.ethz.ch

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
