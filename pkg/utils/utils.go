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

func ExponentialMovingAverage(today uint32, yesterday uint32) uint32 {
	const smoothing float32 = 2
	const days float32 = 5

	frac := smoothing / (1 + days)

	return uint32(float32(today)*frac + float32(yesterday)*(1-frac))
}

// TODO: Make it generic
func ExponentialMovingAverageFloat(today float32, yesterday float32) float32 {
	const smoothing float32 = 2
	const days float32 = 5

	frac := smoothing / (1 + days)

	return float32(float32(today)*frac + float32(yesterday)*(1-frac))
}
