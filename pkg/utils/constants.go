package utils

import "time"

const (
	Localhost string = "0.0.0.0"

	TCP string = "tcp"

	TestDockerImageName string = "docker.io/cvetkovic/dirigent_empty_function:latest"

	TolerateHeartbeatMisses = 3
	HeartbeatInterval       = 5 * time.Second

	WorkerNodeTrafficTimeout = 60 * time.Second

	GRPCConnectionTimeout = 5 * time.Second
	GRPCFunctionTimeout   = 15 * time.Minute // AWS Lambda

	DEFAULT_AUTOSCALER    = "default"
	PREDICTIVE_AUTOSCALER = "predictive"
	MU_AUTOSCALER         = "mu"
)
