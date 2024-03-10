package utils

import (
	"context"
	"github.com/sirupsen/logrus"
	"log"
	"os/signal"
	"os/user"
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

func IsRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		log.Fatalf("[isRoot] Unable to get current user: %s", err)
	}
	return currentUser.Username == "root"
}
