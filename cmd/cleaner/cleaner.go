package main

import (
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

const (
	deployedFunctionName string = "/faas.Executor/Execute"
	controlPlaneAddress  string = "10.10.1.2"
	dataPlaneAddress     string = "10.10.1.3"
)

func Clean() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi := common.InitializeControlPlaneConnection(controlPlaneAddress, utils.DefaultControlPlanePort, -1, -1)

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	_, err := cpApi.ResetMeasurements(ctx, new(emptypb.Empty))
	if err != nil {
		logrus.Errorf("Failed to reset file in control plane : %s", err.Error())
	}

	/*dataplaneApi := common.InitializeDataPlaneConnection(dataPlaneAddress, utils.DefaultDataPlaneProxyPort)

	logrus.Warn("we are here")

	_, err = dataplaneApi.ResetMeasurements(ctx, new(emptypb.Empty))
	if err != nil {
		logrus.Errorf("Failed to reset file in dataplane : %s", err.Error())
	}*/
}

func main() {
	Clean()
}
