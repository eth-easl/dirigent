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
	case "trace":
		logrus.SetLevel(logrus.TraceLevel)
	case "debug":
		logrus.SetLevel(logrus.DebugLevel)
	case "info":
		logrus.SetLevel(logrus.InfoLevel)
	case "warn":
		logrus.SetLevel(logrus.WarnLevel)
	case "error":
		logrus.SetLevel(logrus.ErrorLevel)
	case "fatal":
		logrus.SetLevel(logrus.FatalLevel)
	default:
		logrus.Fatalf("Invalid log level given in the configuarion : %s !", verbosity)
	}
}
