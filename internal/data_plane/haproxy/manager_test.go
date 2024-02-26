package haproxy

import (
	"testing"
)

// TestEditBackend should be executed with sudo
func TestEditBackend(t *testing.T) {
	// To run this test make sure HAProxy is installed on the machine and
	// configs/haproxy.cfg is placed in /etc/haproxy/ folder
	api := NewHAProxyAPI("dummy")

	api.DeleteAllDataplanes()
	if k, _ := api.ListDataplanes(); len(k) != 0 {
		t.Fatal("List of data planes should be empty.")
	}

	api.AddDataplane("127.0.0.1", 8080, true)
	if _, addresses := api.ListDataplanes(); addresses[0] != "127.0.0.1:8080" {
		t.Fatal("Failed to add server 127.0.0.1:8080.")
	}

	api.AddDataplane("127.0.0.1", 8090, true)
	_, addresses := api.ListDataplanes()
	if !((addresses[0] == "127.0.0.1:8080" && addresses[1] == "127.0.0.1:8090") ||
		(addresses[0] == "127.0.0.1:8090" && addresses[1] == "127.0.0.1:8080")) {

		t.Fatal("Failed to add server 127.0.0.1:8090")
	}

	api.ReviseDataplanes([]string{"127.0.0.1:8080", "127.0.0.1:9000"})
	_, addresses = api.ListDataplanes()
	if !((addresses[0] == "127.0.0.1:8080" && addresses[1] == "127.0.0.1:9000") ||
		(addresses[0] == "127.0.0.1:9000" && addresses[1] == "127.0.0.1:8080")) {

		t.Fatal("Failed to revise data plane (throw out 8090 and keep 8080 and 9000)")
	}

	api.RemoveDataplane("127.0.0.1", 8080, true)
	api.RemoveDataplane("127.0.0.1", 9000, true)
	if k, _ := api.ListDataplanes(); len(k) != 0 {
		t.Error("List of data planes should be empty.")
	}
}
