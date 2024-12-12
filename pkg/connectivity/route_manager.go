package connectivity

import (
	"cluster_manager/proto"
	"github.com/sirupsen/logrus"
	"os/exec"
)

type RouteUpdate int

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
	err := exec.Command("sudo", "ip", "route", "add", cidr, "via", gateway).Run()
	if err != nil {
		logrus.Errorf("Failed to install route for %s via %s - %v", cidr, gateway, err)
	}
}

func (rm *RouteManager) deleteRoute(cidr string) {
	err := exec.Command("sudo", "ip", "route", "delete", cidr).Run()
	if err != nil {
		logrus.Errorf("Failed to delete route for %s - %v", cidr, err)
	}
}

func RouteUpdateHandler(rm *RouteManager, update *proto.RouteUpdate) (*proto.ActionStatus, error) {
	switch update.Action {
	case proto.RouteUpdateAction_ROUTE_INSTALL:
		rm.HandleRouteUpdate(RouteInstall, update.CIDR, update.Gateway)
	case proto.RouteUpdateAction_ROUTE_DELETE:
		rm.HandleRouteUpdate(RouteRemove, update.CIDR, update.Gateway) // last parameter irrelevant
	default:
		logrus.Errorf("Unsupported type of action in ReceiveRouteUpdat RPC.")
		return &proto.ActionStatus{Success: false}, nil
	}

	return &proto.ActionStatus{Success: true}, nil
}
