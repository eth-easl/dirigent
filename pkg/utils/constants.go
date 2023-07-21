package utils

import "time"

const (
	Localhost       string = "localhost"
	DockerLocalhost string = "0.0.0.0"

	DefaultDataPlaneProxyPort string = "8080"
	DefaultDataPlaneApiPort   string = "8081"

	DefaultControlPlanePort                    string = "9090"
	DefaultControlPlanePortServiceRegistration string = "9091"

	DefaultWorkerNodePort int = 10010

	DefaultTraceOutputFolder string = "data"

	TCP string = "tcp"

	TestDockerImageName string = "docker.io/cvetkovic/empty_function:latest"

	HeartbeatInterval = 5 * time.Second
)
