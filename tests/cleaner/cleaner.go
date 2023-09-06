package main

import (
	common "cluster_manager/pkg/grpc_helpers"
	"cluster_manager/pkg/utils"
	"cluster_manager/tests/shared"
	"context"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/emptypb"
	"time"
)

func Clean() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	cpApi, err := common.InitializeControlPlaneConnection(shared.ControlPlaneAddress, utils.DefaultControlPlanePort, -1, -1)
	if err != nil {
		logrus.Fatalf("%s", err.Error())
	}

	ctx, cancel := context.WithTimeout(context.Background(), utils.GRPCFunctionTimeout)
	defer cancel()

	_, err = cpApi.ResetMeasurements(ctx, new(emptypb.Empty))
	if err != nil {
		logrus.Errorf("Failed to reset file in control plane : %s", err.Error())
	}
}

func main() {
	Clean()
}
