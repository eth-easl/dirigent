package utils

import (
	"context"
	"github.com/sirupsen/logrus"
	"os/signal"
	"syscall"
)

func WaitTerminationSignal(cleanFunction func()) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		logrus.Info("Received interruption signal, try to gracefully stop")

		cleanFunction()
	}
}
