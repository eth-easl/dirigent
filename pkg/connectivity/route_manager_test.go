package connectivity

import "testing"

func TestRouteInstallAndRemove(t *testing.T) {
	rm := NewRouteManager()
	cidr, defaultGateway := "12.13.0.0/16", "127.0.0.1"

	rm.installRoute(cidr, defaultGateway)
	rm.deleteRoute(cidr)
}
