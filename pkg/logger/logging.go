package logger

import (
	"math/rand"
	"time"

	"github.com/sirupsen/logrus"
)

func SetupLogger(verbosity string) {
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
