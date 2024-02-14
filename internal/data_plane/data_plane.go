package data_plane

import (
	"cluster_manager/api/proto"
	"cluster_manager/internal/data_plane/function_metadata"
	"cluster_manager/internal/data_plane/proxy"
	"cluster_manager/internal/data_plane/proxy/load_balancing"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"path"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Dataplane struct {
	config      config.DataPlaneConfig
	deployments *function_metadata.Deployments
}

func NewDataplane(config config.DataPlaneConfig, deployements *function_metadata.Deployments) *Dataplane {
	return &Dataplane{
		config:      config,
		deployments: deployements,
	}
}

func (d *Dataplane) SetupHeartbeatLoop(proxy proxy.Proxy) {
	for {
		// Send
		d.sendHeartbeatLoop(proxy)

		// Wait
		time.Sleep(utils.HeartbeatInterval)
	}
}

func (d *Dataplane) sendHeartbeatLoop(proxy proxy.Proxy) {
	pollContext, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pollErr := wait.PollUntilContextCancel(pollContext, 5*time.Second, true,
		func(ctx context.Context) (done bool, err error) {
			if proxy == nil {
				return false, errors.New("proxy is null")
			}

			cpApiServer := proxy.GetCpApiServer()
			err = d.registerDataPlane(cpApiServer)
			if err != nil {
				logrus.Errorf("Control plane unreachable. Trying to establish connection with some other replica.")

				cpApiServer, err = grpc_helpers.NewControlPlaneConnection(d.config.ControlPlaneAddress)
				proxy.SetCpApiServer(cpApiServer)

				return false, nil
			}

			return true, nil
		},
	)
	if pollErr != nil {
		logrus.Errorf("Failed to send a heartbeat to the control plane : %s", pollErr)
	} else {
		logrus.Debug("Sent heartbeat to the control plane")
	}
}

func (d *Dataplane) registerDataPlane(cpApiServer proto.CpiInterfaceClient) error {
	grpcPort, _ := strconv.Atoi(d.config.PortGRPC)
	proxyPort, _ := strconv.Atoi(d.config.PortProxy)

	_, err := cpApiServer.RegisterDataplane(context.Background(), &proto.DataplaneInfo{
		IP:        d.config.DataPlaneIp,
		APIPort:   int32(grpcPort),
		ProxyPort: int32(proxyPort),
	})

	return err
}

func (d *Dataplane) AddDeployment(in *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{Success: d.deployments.AddDeployment(in.GetName())}, nil
}

func (d *Dataplane) UpdateEndpointList(patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	if patch.Service == nil && patch.Endpoints == nil {
		d.cleanCache()

		return &proto.DeploymentUpdateSuccess{Success: true}, nil
	}

	deployment, _ := d.deployments.GetDeployment(patch.GetService().GetName())
	if deployment == nil {
		return &proto.DeploymentUpdateSuccess{Success: false}, errors.New("deployment does not exists on the data plane side")
	}

	deployment.AddEndpoints(patch.Endpoints)
	return &proto.DeploymentUpdateSuccess{Success: true}, nil
}

func (d *Dataplane) cleanCache() {
	metadata := d.deployments.ListDeployments()

	for _, fm := range metadata {
		fm.RemoveAllEndpoints()
	}

	logrus.Info("Successfully cleaned the whole cache initiated by the control plane.")
}

func (d *Dataplane) DeleteDeployment(name *proto.ServiceInfo) (*proto.DeploymentUpdateSuccess, error) {
	return &proto.DeploymentUpdateSuccess{Success: d.deployments.DeleteDeployment(name.GetName())}, nil
}

func (d *Dataplane) GetProxyServer(async bool) (proxy.Proxy, error) {
	cpApi, err := grpc_helpers.NewControlPlaneConnection(d.config.ControlPlaneAddress)
	if err != nil {
		return nil, err
	}

	err = d.registerDataPlane(cpApi)
	if err != nil {
		return nil, err
	}

	err = d.syncDeploymentCache(&cpApi, d.deployments)
	if err != nil {
		return nil, err
	}

	loadBalancingPolicy := d.parseLoadBalancingPolicy(d.config)

	if !async {
		return proxy.NewProxyingService(
			d.config.PortProxy,
			d.deployments,
			cpApi,
			path.Join(d.config.TraceOutputFolder, "proxy_trace.csv"),
			loadBalancingPolicy,
		), nil
	} else {
		return proxy.NewAsyncProxyingService(
			d.config,
			d.deployments,
			cpApi,
			path.Join(d.config.TraceOutputFolder, "proxy_trace.csv"),
			loadBalancingPolicy,
		), nil
	}
}

func (d *Dataplane) DeregisterControlPlaneConnection() {
	err := grpc_helpers.DeregisterControlPlaneConnection(&d.config)
	if err != nil {
		logrus.Errorf("Failed to deregister from control plane (error : %s)", err.Error())
	}
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

func (d *Dataplane) syncDeploymentCache(cpApi *proto.CpiInterfaceClient, deployments *function_metadata.Deployments) error {
	resp, err := (*cpApi).ListServices(context.Background(), &emptypb.Empty{})
	if err != nil {
		return errors.New("initial deployment cache synchronization failed")
	}

	for i := 0; i < len(resp.Service); i++ {
		deployments.AddDeployment(resp.Service[i])
	}

	return nil
}

func (d *Dataplane) DrainSandbox(patch *proto.DeploymentEndpointPatch) (*proto.DeploymentUpdateSuccess, error) {
	deployment, _ := d.deployments.GetDeployment(patch.GetService().GetName())
	if deployment == nil {
		return &proto.DeploymentUpdateSuccess{Success: false}, errors.New("deployment does not exists on the data plane side")
	}

	err := deployment.DrainEndpoints(patch.Endpoints)
	return &proto.DeploymentUpdateSuccess{Success: true}, err
}
