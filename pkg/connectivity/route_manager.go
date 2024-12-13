package connectivity

import (
	"cluster_manager/proto"
	"github.com/sirupsen/logrus"
	"os/exec"
)

type RouteUpdate int

const RoutingTableName = "dirigent"

const (
	RouteInstall RouteUpdate = iota
	RouteRemove
)

type Route struct {
	CIDR    string
	Gateway string
}

type RouteManager struct {
	routes []Route
}

func NewRouteManager() *RouteManager {
	return &RouteManager{}
}

func (rm *RouteManager) HandleRouteUpdate(action RouteUpdate, cidr string, gateway string) {
	switch action {
	case RouteInstall:
		rm.installRoute(cidr, gateway)
	case RouteRemove:
		rm.deleteRoute(cidr)
	default:
		logrus.Errorf("Unsupported route update operation.")
	}
}

func (rm *RouteManager) installRoute(cidr string, gateway string) {
	err := exec.Command("sudo", "ip", "route", "add", cidr, "via", gateway, "table", RoutingTableName).Run()
	if err != nil {
		logrus.Errorf("Failed to install route for %s via %s - %v", cidr, gateway, err)
	}
}

func (rm *RouteManager) deleteRoute(cidr string) {
	err := exec.Command("sudo", "ip", "route", "delete", cidr, "table", RoutingTableName).Run()
	if err != nil {
		logrus.Errorf("Failed to delete route for %s - %v", cidr, err)
	}
}

func RouteUpdateHandler(rm *RouteManager, update *proto.RouteUpdate) (*proto.ActionStatus, error) {
	for _, route := range update.Routes {
		switch update.Action {
		case proto.RouteUpdateAction_ROUTE_INSTALL:
			rm.HandleRouteUpdate(RouteInstall, route.CIDR, route.Gateway)
		case proto.RouteUpdateAction_ROUTE_DELETE:
			rm.HandleRouteUpdate(RouteRemove, route.CIDR, route.Gateway) // last parameter irrelevant
		default:
			logrus.Errorf("Unsupported type of action in ReceiveRouteUpdat RPC.")
			return &proto.ActionStatus{Success: false}, nil
		}
	}

	return &proto.ActionStatus{Success: true}, nil
}

func RouteActionToString(action RouteUpdate) string {
	switch action {
	case RouteInstall:
		return "RouteInstall"
	case RouteRemove:
		return "RouteRemove"
	default:
		logrus.Fatal("Unsupported RouteAction type for parsing to string.")
		return ""
	}
}
