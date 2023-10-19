package main

import (
	"cluster_manager/tests/shared"
	"fmt"
	"github.com/sirupsen/logrus"
	"time"
)

const (
	ControlPlaneAddress string = "localhost"
	MockIp              string = "mockIP"
)

func main() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	times := 0

	for times < 1 {
		counter := 0
		nbWorkers := 1
		// Measures
		for nbWorkers <= 10000 {
			start := time.Now()

			shared.DeployDataplanes(nbWorkers, counter)

			delta := time.Since(start)
			fmt.Printf("%d,", delta)

			counter += nbWorkers
			nbWorkers *= 10
		}
		fmt.Println()

		times++
	}

}
