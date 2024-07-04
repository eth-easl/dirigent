package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer"
	"cluster_manager/internal/control_plane/control_plane/endpoint_placer/eviction_policy"
	placement_policy2 "cluster_manager/internal/control_plane/control_plane/endpoint_placer/placement_policy"
	"cluster_manager/internal/control_plane/control_plane/image_storage"
	"cluster_manager/internal/control_plane/control_plane/service_state"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/synchronization"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"math/rand"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

/*
Helpers for control_plane.go
*/

// Only one goroutine will execute the function per service
func (c *ControlPlane) notifyDataplanesAndStartScalingLoop(ctx context.Context, serviceInfo *proto.ServiceInfo) error {

	c.DataPlaneConnections.Lock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		_, err := conn.AddDeployment(ctx, serviceInfo)
		if err != nil {
			logrus.Warnf("Failed to add deployment to data plane %s - %v", conn.GetIP(), err)
		}
	}
	c.DataPlaneConnections.Unlock()

	serviceState := service_state.NewFunctionState(serviceInfo)

	c.autoscalingManager.Create(serviceState)

	placementPolicy, evictionPolicy := parsePlacementEvictionPolicies(c.Config)

	c.SIStorage.Set(serviceInfo.Name, &endpoint_placer.EndpointPlacer{
		ServiceState:            serviceState,
		ColdStartTracingChannel: c.ColdStartTracing.InputChannel,
		PlacementPolicy:         placementPolicy,
		EvictionPolicy:          evictionPolicy,
		PersistenceLayer:        c.PersistenceLayer,
		NIStorage:               c.NIStorage,
		ImageStorage:            c.imageStorage,
		DataPlaneConnections:    c.DataPlaneConnections,
		DandelionNodes:          synchronization.NewControlPlaneSyncStructure[string, bool](),
	})

	go c.SIStorage.GetNoCheck(serviceInfo.Name).ScalingControllerLoop()

	return nil
}

func (c *ControlPlane) removeServiceFromDataplane(ctx context.Context, serviceInfo *proto.ServiceInfo) error {
	c.DataPlaneConnections.Lock()
	for _, conn := range c.DataPlaneConnections.GetMap() {
		_, err := conn.DeleteDeployment(ctx, serviceInfo)
		if err != nil {
			return err
		}
	}
	c.DataPlaneConnections.Unlock()

	return nil
}

func (c *ControlPlane) removeEndpointsAssociatedWithNode(nodeID string) {
	c.SIStorage.Lock()
	for _, value := range c.SIStorage.GetMap() {
		toExclude := make(map[*core.Endpoint]struct{})
		value.ServiceState.EndpointLock.Lock()
		for _, endpoint := range value.ServiceState.Endpoints {
			if endpoint.Node.GetName() == nodeID {
				toExclude[endpoint] = struct{}{}
			}
		}

		value.ExcludeEndpoints(toExclude)
		value.ServiceState.EndpointLock.Unlock()

		atomic.AddInt64(&value.ServiceState.ActualScale, int64(-len(toExclude)))
	}
	c.SIStorage.Unlock()
}

func pickNodeToPrepull(url string, images image_storage.ImageStorage, nodes synchronization.SyncStructure[string, core.WorkerNodeInterface]) (core.WorkerNodeInterface, error) {
	images.Lock()
	defer images.Unlock()
	if imageInfo, ok := images.Get(url); ok && imageInfo.Count > 0 {
		// Image already prepulled, nothing to prepull.
		return nil, nil
	}
	allNodes := nodes.GetValues()
	if len(allNodes) == 0 {
		// No nodes available to prepull.
		return nil, nil
	}
	node := allNodes[rand.Intn(len(allNodes))]
	err := images.RegisterWithFetch(url, node)
	return node, err
}

func prepullOneImage(ctx context.Context, node core.WorkerNodeInterface, imageInfo *proto.ImageInfo) error {
	ctx, cancel := context.WithTimeout(ctx, utils.WorkerNodeTrafficTimeout)
	defer cancel()
	_, err := node.PrepullImage(ctx, imageInfo)
	if err != nil {
		logrus.Warnf("Failed to prepull image %s on node %s (error: %s)", imageInfo.URL, node.GetName(), err.Error())
	}
	return err
}

func prepullMultipleImages(ctx context.Context, node core.WorkerNodeInterface, imageInfo *proto.ImageInfo, images image_storage.ImageStorage) error {
	_, err := node.PrepullImage(ctx, imageInfo)
	if err != nil {
		logrus.Warnf("Failed to prepull image on node %s (error: %s)", node.GetName(), err.Error())
		return err
	}
	images.Lock()
	defer images.Unlock()
	err = images.RegisterWithFetch(imageInfo.URL, node)
	if err != nil {
		logrus.Warnf("Failed to register image %s on node %s (error: %s)", imageInfo.URL, node.GetName(), err.Error())
	}
	return err
}

func prepullOneImageSync(ctx context.Context, imageInfo *proto.ImageInfo, images image_storage.ImageStorage, nodes synchronization.SyncStructure[string, core.WorkerNodeInterface]) error {
	node, err := pickNodeToPrepull(imageInfo.URL, images, nodes)
	if err != nil {
		return err
	}
	if node == nil {
		return nil
	}
	return prepullOneImage(ctx, node, imageInfo)
}

func prepullOneImageAsync(imageInfo *proto.ImageInfo, images image_storage.ImageStorage, nodes synchronization.SyncStructure[string, core.WorkerNodeInterface]) {
	node, err := pickNodeToPrepull(imageInfo.URL, images, nodes)
	if err != nil {
		logrus.Warnf("Failed to pick a node to prepull image %s to: %s", imageInfo.URL, err.Error())
		return
	}
	if node == nil {
		return
	}
	go prepullOneImage(context.Background(), node, imageInfo)
}

func prepullAllImagesSync(ctx context.Context, imageInfo *proto.ImageInfo, images image_storage.ImageStorage, nodes synchronization.SyncStructure[string, core.WorkerNodeInterface]) error {
	eg := &errgroup.Group{}
	for _, node := range nodes.GetMap() {
		currNode := node
		eg.Go(func() error {
			return prepullMultipleImages(ctx, currNode, imageInfo, images)
		})
	}
	return eg.Wait()
}

func prepullAllImagesAsync(imageInfo *proto.ImageInfo, images image_storage.ImageStorage, nodes synchronization.SyncStructure[string, core.WorkerNodeInterface]) {
	ctx := context.Background()
	for _, node := range nodes.GetMap() {
		go prepullMultipleImages(ctx, node, imageInfo, images)
	}
}

func (c *ControlPlane) prepullImagesForService(ctx context.Context, mode proto.PrepullMode, url string) error {
	imageInfo := &proto.ImageInfo{URL: url}
	switch mode {
	case proto.PrepullMode_ONE_SYNC:
		return prepullOneImageSync(ctx, imageInfo, c.imageStorage, c.NIStorage)
	case proto.PrepullMode_ONE_ASYNC:
		prepullOneImageAsync(imageInfo, c.imageStorage, c.NIStorage)
	case proto.PrepullMode_ALL_SYNC:
		return prepullAllImagesSync(ctx, imageInfo, c.imageStorage, c.NIStorage)
	case proto.PrepullMode_ALL_ASYNC:
		prepullAllImagesAsync(imageInfo, c.imageStorage, c.NIStorage)
	}
	return nil
}

func searchEndpointByContainerName(endpoints []*core.Endpoint, cid string) *core.Endpoint {
	for _, e := range endpoints {
		if e.SandboxID == cid {
			return e
		}
	}

	return nil
}

/*
* Functions used for testing
 */

func (c *ControlPlane) GetNumberConnectedWorkers() int {
	return c.NIStorage.AtomicLen()
}

func (c *ControlPlane) GetNumberDataplanes() int {
	return c.DataPlaneConnections.AtomicLen()
}

func (c *ControlPlane) GetNumberServices() int {
	return c.SIStorage.AtomicLen()
}

// Parse placement and eviction policies

func parsePlacementEvictionPolicies(controlPlaneConfig *config.ControlPlaneConfig) (placement_policy2.PlacementPolicy, eviction_policy.EvictionPolicy) {
	var placementPolicy placement_policy2.PlacementPolicy
	switch controlPlaneConfig.PlacementPolicy {
	case "random":
		placementPolicy = placement_policy2.NewRandomPlacement()
	case "round-robin":
		placementPolicy = placement_policy2.NewRoundRobinPlacement()
	case "kubernetes":
		placementPolicy = placement_policy2.NewKubernetesPolicy()
	case "dandelion":
		placementPolicy = placement_policy2.NewDandelionPlacementPolicy()
	default:
		logrus.Error("Failed to parse placement, default policy is random")
		placementPolicy = placement_policy2.NewRandomPlacement()
	}

	return placementPolicy, eviction_policy.NewDefaultevictionPolicy()
}

func isValidAutoscaler(autoscaler string) bool {
	return autoscaler == utils.DEFAULT_AUTOSCALER || autoscaler == utils.PREDICTIVE_AUTOSCALER || autoscaler == utils.MU_AUTOSCALER
}
