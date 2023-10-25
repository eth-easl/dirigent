package main

import (
	"cluster_manager/tests/shared"
	"flag"
	"github.com/sirupsen/logrus"
	"time"
)

var (
	invocations = flag.Int("invocations", 1, "number of invocations")
)

const (
	SECOND = 1e9
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	flag.Parse()

	logrus.Info("Registering services")
	shared.DeployServiceMultiThread(*invocations, 0)

	logrus.Info("Starting burst test")

	shared.PerformXInvocations(*invocations, 0)

	logrus.Info("End burst test")
}
