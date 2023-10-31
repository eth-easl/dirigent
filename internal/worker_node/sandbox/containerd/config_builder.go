package containerd

import (
	"cluster_manager/api/proto"
	"math/rand"
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
