package tests

import (
	utils2 "cluster_manager/tests/utils"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	ZERO_OFFSET int = 0
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	fmt.Println(string(b))
	return string(b)
}

func TestDeployRandomService(t *testing.T) {
	utils2.DeployService(t, 1, ZERO_OFFSET)
}

func TestDeploy10Services(t *testing.T) {
	utils2.DeployService(t, 10, ZERO_OFFSET)
}

func TestDeploy100Services(t *testing.T) {
	utils2.DeployService(t, 100, ZERO_OFFSET)
}

func TestDeploy1000Services(t *testing.T) {
	utils2.DeployService(t, 1000, ZERO_OFFSET)
}

func TestDeploy10000Services(t *testing.T) {
	utils2.DeployService(t, 10000, ZERO_OFFSET)
}

func TestDeploy100000Services(t *testing.T) {
	utils2.DeployService(t, 100000, ZERO_OFFSET)
}

func TestInvocationProxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	utils2.PerformXInvocations(t, 1, ZERO_OFFSET)
}

func TestInvocation25Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	utils2.PerformXInvocations(t, 25, ZERO_OFFSET)
}

func TestInvocation50Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	utils2.PerformXInvocations(t, 50, ZERO_OFFSET)
}

func TestInvocation100Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	utils2.PerformXInvocations(t, 100, ZERO_OFFSET)
}

func TestInvocation200Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	utils2.PerformXInvocations(t, 200, ZERO_OFFSET)
}

func TestInvocation400Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	utils2.PerformXInvocations(t, 400, ZERO_OFFSET)
}

func TestInvocation800Proxying(t *testing.T) {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	utils2.PerformXInvocations(t, 800, ZERO_OFFSET)
}
