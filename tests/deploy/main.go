package main

import (
	"cluster_manager/tests/utils"
	"flag"
	"testing"

	"github.com/sirupsen/logrus"
)

var (
	nbServices = flag.Int("invocations", 1, "number of invocations")
	offset     = flag.Int("offset", 0, "offset")
)

func main() {
	flag.Parse()

	logrus.Info("Registering services")
	utils.DeployService(&testing.T{}, *nbServices, *offset)
	logrus.Info("End deploy services")
}
