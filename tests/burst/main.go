package main

import (
	"cluster_manager/tests/shared"
	"flag"
	"testing"

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
	shared.DeployService(&testing.T{}, *invocations, 0)

	logrus.Info("Starting burst test")

	shared.PerformXInvocations(&testing.T{}, *invocations, 0)

	logrus.Info("End burst test")
}
