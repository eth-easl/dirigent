package sandbox

import (
	"cluster_manager/api/proto"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"math/rand"
	"strconv"
	"sync"
)

var portsInUse = make(map[int]struct{})
var portsInUseMutex = sync.Mutex{}

func l4ProtocolToString(protocol proto.L4Protocol) string {
	switch protocol.Number() {
	case 1:
		return "udp"
	default:
		return "tcp"
	}
}

func validatePort(port int32) bool {
	if port <= 0 || port >= 65536 {
		return false
	}

	return true
}

func AssignRandomPort() int {
	portsInUseMutex.Lock()
	defer portsInUseMutex.Unlock()

	var port int

	for {
		port = 1024 + rand.Intn(64000)

		_, ok := portsInUse[port]
		if !ok {
			portsInUse[port] = struct{}{}

			break
		}
	}

	return port
}

func UnassignPort(port int) {
	portsInUseMutex.Lock()
	defer portsInUseMutex.Unlock()

	delete(portsInUse, port)
}

func createPortMappings(r *proto.PortMapping) (nat.PortMap, nat.PortSet, error) {
	host := make(nat.PortMap)
	guest := make(nat.PortSet)

	if !validatePort(r.GuestPort) {
		return nil, nil, errors.New("Invalid service info port mapping specification.")
	}

	common := nat.Port(fmt.Sprintf("%d/%s", r.GuestPort, l4ProtocolToString(r.Protocol)))

	hostPort := AssignRandomPort()

	r.HostPort = int32(hostPort)
	host[common] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0", // TODO: hardcoded while VXLAN not implemented
			HostPort: strconv.Itoa(hostPort),
		},
	}
	guest[common] = struct{}{}

	return host, guest, nil
}

func CreateSandboxConfig(image string, portMaps *proto.PortMapping) (*container.HostConfig, *container.Config, error) {
	hostMapping, guestMapping, err := createPortMappings(portMaps)
	if err != nil {
		return nil, nil, err
	}

	hostConfig := &container.HostConfig{PortBindings: hostMapping}
	containerConfig := &container.Config{Image: image, ExposedPorts: guestMapping}

	return hostConfig, containerConfig, nil
}
