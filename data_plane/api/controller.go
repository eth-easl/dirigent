package api

import (
	"cluster_manager/common"
	"math/rand"
)

type PFStateController struct {
	ActualScale  int
	DesiredScale int

	DesiredStateChannel *chan int
}

func placementPolicy(storage *NodeInfoStorage) *WorkerNode {
	storage.Lock()
	defer storage.Unlock()

	nodes := common.Keys(storage.NodeInfo)
	index := rand.Intn(len(nodes))

	nodeName := nodes[index]
	nodeRef := storage.NodeInfo[nodeName]

	return nodeRef
}

func evictionPolicy(endpoint *[]Endpoint) (*Endpoint, []Endpoint) {
	index := rand.Intn(len(*endpoint))
	newEndpointList := append((*endpoint)[:index], (*endpoint)[index+1:]...)

	return &(*endpoint)[index], newEndpointList
}
