package utils

import (
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/constraints"
	"log"
	"os/signal"
	"os/user"
	"syscall"
)

type Number interface {
	constraints.Integer | constraints.Float
}

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

func ExponentialMovingAverage[T Number](today T, yesterday T) T {
	return T(float32(today)*0.8 + float32(yesterday)*0.2)
}
