package main

import (
	"cluster_manager/tests/utils"
	"flag"
	"github.com/sirupsen/logrus"
	"testing"
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
	utils.DeployService(&testing.T{}, *invocations, 0)

	logrus.Info("Starting burst test")

	utils.PerformXInvocations(&testing.T{}, *invocations, 0)

	logrus.Info("End burst test")
}
