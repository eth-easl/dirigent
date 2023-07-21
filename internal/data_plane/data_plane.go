package data_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"path"
	"strconv"
)

type Dataplane struct {
	config       config.DataPlaneConfig
	deployements *function_metadata.Deployments
}

func NewDataplane(config config.DataPlaneConfig, deployements *function_metadata.Deployments) *Dataplane {
	return &Dataplane{
		config:       config,
		deployements: deployements,
	}
}

func (d *Dataplane) AddDeployment(in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{Success: d.deployements.AddDeployment(in.GetName())}, nil
}

func (d *Dataplane) UpdateEndpointList(patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	deployment, _ := d.deployements.GetDeployment(patch.GetService().GetName())
	if deployment == nil {
		return &proto.DeploymentUpdateSuccess{Success: false}, nil
	}

	err := deployment.SetUpstreamURLs(patch.Endpoints)
	if err != nil {
		return nil, err
	}

	return &proto.DeploymentUpdateSuccess{Success: true}, nil
}

func (d *Dataplane) DeleteDeployment(name *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{Success: d.deployements.DeleteDeployment(name.GetName())}, nil
}

func (d *Dataplane) GetProxyServer() *proxy.ProxyingService {
	var dpConnection proto.CpiInterfaceClient

	grpcPort, _ := strconv.Atoi(d.config.PortGRPC)
	proxyPort, _ := strconv.Atoi(d.config.PortProxy)

	dpConnection = grpc_helpers.InitializeControlPlaneConnection(d.config.ControlPlaneIp, d.config.ControlPlanePort, int32(grpcPort), int32(proxyPort))

	d.syncDeploymentCache(&dpConnection, d.deployements)

	loadBalancingPolicy := d.parseLoadBalancingPolicy(d.config)

	return proxy.NewProxyingService("0.0.0.0", d.config.PortProxy, d.deployements, &dpConnection, path.Join(d.config.TraceOutputFolder, "proxy_trace.csv"), loadBalancingPolicy)
}

func (d *Dataplane) DeregisterControlPlaneConnection() {
	grpcPort, _ := strconv.Atoi(d.config.PortGRPC)
	proxyPort, _ := strconv.Atoi(d.config.PortProxy)

	grpc_helpers.DeregisterControlPlaneConnection(d.config.ControlPlaneIp, d.config.ControlPlanePort, int32(grpcPort), int32(proxyPort))
}

func (d *Dataplane) parseLoadBalancingPolicy(dataPlaneConfig config.DataPlaneConfig) load_balancing.LoadBalancingPolicy {
	switch dataPlaneConfig.LoadBalancingPolicy {
	case "random":
		return load_balancing.LOAD_BALANCING_RANDOM
	case "round-robin":
		return load_balancing.LOAD_BALANCING_ROUND_ROBIN
	case "least-processed":
		return load_balancing.LOAD_BALANCING_LEAST_PROCESSED
	case "knative":
		return load_balancing.LOAD_BALANCING_KNATIVE
	default:
		logrus.Error("Failed to parse policy, default policy is random")
		return load_balancing.LOAD_BALANCING_RANDOM
	}
}

func (d *Dataplane) syncDeploymentCache(cpApi *proto.CpiInterfaceClient, deployments *function_metadata.Deployments) {
	resp, err := (*cpApi).ListServices(context.Background(), &emptypb.Empty{})
	if err != nil {
		logrus.Fatal("Initial deployment cache synchronization failed.")
	}

	for i := 0; i < len(resp.Service); i++ {
		deployments.AddDeployment(resp.Service[i])
	}
}
