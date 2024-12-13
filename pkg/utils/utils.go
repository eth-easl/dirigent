package utils

import (
	"context"
	"github.com/sirupsen/logrus"
	"golang.org/x/exp/constraints"
	"log"
	"os"
	"os/signal"
	"os/user"
	"strings"
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

	if currentUser.Username != "root" {
		logrus.Errorf("Worker node must be started with sudo.")
	}

	return currentUser.Username == "root"
}

func ExponentialMovingAverage[T Number](today T, yesterday T) T {
	return T(float32(today)*0.8 + float32(yesterday)*0.2)
}

func SysctlCheck(path string, expectedValue string) bool {
	b, err := os.ReadFile(path)
	if err != nil {
		logrus.Errorf("Failed to read from %s.", path)
	}

	data := strings.TrimSpace(string(b))
	if data != expectedValue {
		logrus.Errorf("Dirigent cannot operate without %s enabled. Make sure to persist this setting over restarts.", path)
	}

	return data == expectedValue
}

func IsIpV4ForwardingEnabled() bool {
	return SysctlCheck("/proc/sys/net/ipv4/ip_forward", "1")
}

func IsRouteLocalnetEnabled() bool {
	return SysctlCheck("/proc/sys/net/ipv4/conf/all/route_localnet", "1")
}

func PassesWorkerNodeConfigurationChecks() bool {
	return IsRoot() &&
		IsIpV4ForwardingEnabled() &&
		IsRouteLocalnetEnabled()
}
