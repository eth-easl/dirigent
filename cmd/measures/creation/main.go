package main

import (
	"cluster_manager/tests/shared"
	"github.com/sirupsen/logrus"
	"time"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	nbServices := 150000
	shared.DeployWorkers(5, 0)
	shared.DeployServiceMultiThread(nbServices, 0)

	for j := 0; j < 1; j++ {
		shared.FireColdstartMultiThread(nbServices, 0, 1)
		time.Sleep(1 * time.Second)
		shared.FireColdstartMultiThread(nbServices, 0, 0)
		time.Sleep(time.Second)
	}

}
