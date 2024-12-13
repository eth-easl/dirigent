package routing

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/pkg/connectivity"
	"cluster_manager/proto"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

type RouteUpdateGRPC func(context.Context, *proto.RouteUpdate, ...grpc.CallOption) (*proto.ActionStatus, error)

func CreateRoute(cidr string, gateway string) *proto.Route {
	return &proto.Route{CIDR: cidr, Gateway: gateway}
}

func ExtractRoutes(wni []core.WorkerNodeInterface) []*proto.Route {
	var routes []*proto.Route
	for _, wn := range wni {
		routes = append(routes, CreateRoute(wn.GetCIDR(), wn.GetIP()))
	}

	return routes
}

func routeUpdateActionToProto(update connectivity.RouteUpdate) proto.RouteUpdateAction {
	switch update {
	case connectivity.RouteInstall:
		return proto.RouteUpdateAction_ROUTE_INSTALL
	case connectivity.RouteRemove:
		return proto.RouteUpdateAction_ROUTE_DELETE
	default:
		logrus.Fatalf("Unsupported type conversion in RouteUpdateAction to gRPC type.")
	}

	return proto.RouteUpdateAction_ROUTE_INSTALL
}

func SendRoutes(f RouteUpdateGRPC, ctx context.Context, action connectivity.RouteUpdate, routes []*proto.Route, componentName string) {
	status, err := f(ctx, &proto.RouteUpdate{
		Action: routeUpdateActionToProto(action),
		Routes: routes,
	})

	if err != nil || !status.Success {
		logrus.Errorf("Error for %v action for %v with %s - %v", connectivity.RouteActionToString(action), routes, componentName, err)
	}

	logrus.Debugf("Successfully instructed %v action for %v to %s.", connectivity.RouteActionToString(action), routes, componentName)
}
