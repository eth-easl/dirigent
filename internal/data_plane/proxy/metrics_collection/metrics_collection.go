package metrics_collection

import (
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"github.com/sirupsen/logrus"
	"time"
)

const number_minutes int = 60

type DurationInvocation struct {
	Duration    uint32
	ServiceName string
}

type MetadataPerFunction struct {
	invocationPerSeconds []float64
	averageDuration      uint32
}

type MetricsCollector struct {
	controlPlane proto.CpiInterfaceClient

	metadata map[string]*MetadataPerFunction

	controlPlaneNotifyInterval int

	sendChannel   *time.Ticker
	updateChannel *time.Ticker

	incomingRequestChannel chan string
	doneRequestChannel     chan DurationInvocation
}

func NewMetricsCollector(controlPlaneNotifyInterval int, cp proto.CpiInterfaceClient) (chan string, chan DurationInvocation) {
	metricsCollector := &MetricsCollector{
		controlPlane: cp,

		metadata: make(map[string]*MetadataPerFunction),

		controlPlaneNotifyInterval: controlPlaneNotifyInterval,

		sendChannel:            time.NewTicker(time.Duration(controlPlaneNotifyInterval) * time.Minute),
		updateChannel:          time.NewTicker(time.Minute),
		incomingRequestChannel: make(chan string, 1000),
		doneRequestChannel:     make(chan DurationInvocation, 1000),
	}

	go metricsCollector.metricGatheringLoop()

	return metricsCollector.incomingRequestChannel, metricsCollector.doneRequestChannel
}

func (c *MetricsCollector) metricGatheringLoop() {
	for {
		select {
		case <-c.sendChannel.C:
			c.sendMetricsToControlPlane()
		case <-c.updateChannel.C:
			c.shiftBins()
		case service := <-c.incomingRequestChannel:
			logrus.Tracef("Received data from incomingRequestChannel : %s", service)
			if _, ok := c.metadata[service]; !ok {
				c.metadata[service] = &MetadataPerFunction{
					invocationPerSeconds: make([]float64, 60),
					averageDuration:      0,
				}
			}

			currentSlice := c.metadata[service]
			currentSlice.invocationPerSeconds[0]++
		case duration := <-c.doneRequestChannel:
			logrus.Tracef("Received data from doneRequestChannel : %s", duration.ServiceName)
			// TODO: Clean this
			if tmp, ok := c.metadata[duration.ServiceName]; ok {
				tmp.averageDuration = utils.ExponentialMovingAverage[uint32](duration.Duration, tmp.averageDuration)
			}
		}
	}
}

func (c *MetricsCollector) sendMetricsToControlPlane() {
	logrus.Infof("Sending values in the control plane")

	c.controlPlane.SetBackgroundMetrics(context.Background(), c.parseMetrics())
}

func (c *MetricsCollector) parseMetrics() *proto.MetricsPredictiveAutoscaler {
	metricsPerFunctions := make([]*proto.FunctionMetrics, 0, len(c.metadata))

	for functionName, values := range c.metadata {
		metricsPerFunctions = append(metricsPerFunctions, &proto.FunctionMetrics{
			FunctionName:         functionName,
			FunctionDuration:     values.averageDuration,
			InvocationsPerMinute: values.invocationPerSeconds,
		})
	}

	return &proto.MetricsPredictiveAutoscaler{
		Metric: metricsPerFunctions,
	}
}

func (c *MetricsCollector) shiftBins() {
	for _, value := range c.metadata {
		for i := number_minutes - 1; i > 0; i-- {
			value.invocationPerSeconds[i] = value.invocationPerSeconds[i-1]
		}
		value.invocationPerSeconds[0] = 0
	}
}
