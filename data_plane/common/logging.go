package common

import (
	"github.com/sirupsen/logrus"
	"math/rand"
	"time"
)

func InitLibraries(verbosity string) {
	rand.Seed(time.Now().UnixNano())

	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})
	switch verbosity {
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	default:
		logrus.SetLevel(logrus.InfoLevel)
	}
}
