package main

import (
	"cluster_manager/tests/shared"
	"flag"
	"github.com/sirupsen/logrus"
)

var (
	invocations = flag.Int("invocations", 1, "number of invocations")
)

const (
	SECOND = 1e9
)

func main() {
	flag.Parse()

	logrus.Info("Registering services")
	shared.DeployService(*invocations, 0)

	logrus.Info("Starting burst test")

	shared.PerformXInvocations(*invocations, 0)

	logrus.Info("End burst test")
}
