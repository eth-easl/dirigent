package connectivity

import (
	"cluster_manager/proto"
	"github.com/sirupsen/logrus"
	"os/exec"
)

type RouteUpdate int

const ipRulePriority = "1"
const ipRouteTableName = "dirigent"

const (
	RouteInstall RouteUpdate = iota
	RouteRemove
)

type Route struct {
	CIDR    string
	Gateway string
}

type RouteManager struct{}

func NewRouteManager() *RouteManager {
	return &RouteManager{}
}

func (rm *RouteManager) HandleRouteUpdate(action RouteUpdate, cidr string, gateway string) {
	switch action {
	case RouteInstall:
		logrus.Debugf("Installing route for %s via %s.", cidr, gateway)
		rm.installRoute(cidr, gateway)
	case RouteRemove:
		logrus.Debugf("Deleting route for %s.", cidr)
		rm.deleteRoute(cidr)
	default:
		logrus.Errorf("Unsupported route update operation.")
	}
}

func (rm *RouteManager) installRoute(cidr string, gateway string) {
	err := exec.Command("sudo", "ip", "rule", "add", "to", cidr, "lookup", ipRouteTableName, "priority", ipRulePriority).Run()
	if err != nil {
		logrus.Errorf("Failed to install IP rule for %s - %v", cidr, err)
	}

	err = exec.Command("sudo", "ip", "route", "add", cidr, "via", gateway, "table", ipRouteTableName).Run()
	if err != nil {
		logrus.Errorf("Failed to install IP route for %s via %s - %v", cidr, gateway, err)
	}
}

func (rm *RouteManager) deleteRoute(cidr string) {
	err := exec.Command("sudo", "ip", "rule", "delete", "to", cidr, "lookup", ipRouteTableName, "priority", ipRulePriority).Run()
	if err != nil {
		logrus.Errorf("Failed to delete rule for %s - %v", cidr, err)
	}

	err = exec.Command("sudo", "ip", "route", "delete", cidr, "table", ipRouteTableName).Run()
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

func flushIpRules() {
	for {
		err := exec.Command("sudo", "ip", "rule", "delete", "lookup", ipRouteTableName).Run()
		if err != nil {
			break
		}
	}
}

func FlushIPRoutes() {
	flushIpRules()

	err := exec.Command("sudo", "ip", "route", "flush", "table", ipRouteTableName).Run()
	if err != nil {
		logrus.Errorf("Error flushing IP route table Dirigent - %v", err.Error())
	}
}
