package sandbox

import (
	"cluster_manager/api/proto"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/go-connections/nat"
	"math/rand"
	"strconv"
)

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

func assignRandomPort() int {
	return 1024 + rand.Intn(4096)
}

func createPortMappings(rawMappings []*proto.PortMapping) (nat.PortMap, nat.PortSet, error) {
	host := make(nat.PortMap)
	guest := make(nat.PortSet)

	for _, r := range rawMappings {
		if !validatePort(r.GuestPort) {
			return nil, nil, errors.New("Invalid service info port mapping specification.")
		}

		common := nat.Port(fmt.Sprintf("%d/%s", r.GuestPort, l4ProtocolToString(r.Protocol)))

		host[common] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0", // TODO: hardcoded while VXLAN not implemented
				HostPort: strconv.Itoa(assignRandomPort()),
			},
		}
		guest[common] = struct{}{}
	}

	return host, guest, nil
}

func CreateSandboxConfig(image string, portMaps []*proto.PortMapping) (*container.HostConfig, *container.Config, error) {
	hostMapping, guestMapping, err := createPortMappings(portMaps)
	if err != nil {
		return nil, nil, err
	}

	hostConfig := &container.HostConfig{
		PortBindings: hostMapping,
	}

	containerConfig := &container.Config{
		Image:        image,
		ExposedPorts: guestMapping,
	}

	return hostConfig, containerConfig, nil
}
