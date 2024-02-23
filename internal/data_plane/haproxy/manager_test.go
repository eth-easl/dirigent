package haproxy

import (
	"testing"
)

func TestEditBackend(t *testing.T) {
	api := NewHAProxyAPI()

	api.DeleteAllDataplanes()
	if k, _ := api.ListDataplanes(); len(k) != 0 {
		t.Fatal("List of data planes should be empty.")
	}

	api.AddDataplane("127.0.0.1", 8080)
	if _, addresses := api.ListDataplanes(); addresses[0] != "127.0.0.1:8080" {
		t.Fatal("Failed to add server 127.0.0.1:8080.")
	}

	api.AddDataplane("127.0.0.1", 8090)
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

	api.RemoveDataplane("127.0.0.1", 8080)
	api.RemoveDataplane("127.0.0.1", 9000)
	if k, _ := api.ListDataplanes(); len(k) != 0 {
		t.Error("List of data planes should be empty.")
	}
}
