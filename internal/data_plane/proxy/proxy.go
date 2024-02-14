package proxy

import "cluster_manager/api/proto"

type Proxy interface {
	StartProxyServer()
	StartTracingService()

	GetCpApiServer() proto.CpiInterfaceClient
	SetCpApiServer(client proto.CpiInterfaceClient)
}
