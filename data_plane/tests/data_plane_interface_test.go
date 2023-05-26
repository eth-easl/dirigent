package tests

import (
	"testing"
)

func TestInstanceAppeared(t *testing.T) {
	UpdateEndpointList(t, "localhost", "8081", []string{"localhost:10000"})
}
