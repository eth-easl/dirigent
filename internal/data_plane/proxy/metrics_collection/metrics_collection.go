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
	invocationPerSeconds []uint32
	averageDuration      uint32
}

type MetricsCollector struct {
	controlPlane proto.CpiInterfaceClient

	metadata map[string]*MetadataPerFunction

	controlPlaneNotifyInterval int

	sendChannel   chan struct{}
	updateChannel chan struct{}

	incomingRequestChannel chan string
	doneRequestChannel     chan DurationInvocation
}

func NewMetricsCollector(controlPlaneNotifyInterval int, cp proto.CpiInterfaceClient) (chan string, chan DurationInvocation) {
	metricsCollector := &MetricsCollector{
		controlPlane: cp,

		metadata: make(map[string]*MetadataPerFunction),

		controlPlaneNotifyInterval: controlPlaneNotifyInterval,

		sendChannel:            make(chan struct{}, 1000),
		updateChannel:          make(chan struct{}, 1000),
		incomingRequestChannel: make(chan string, 1000),
		doneRequestChannel:     make(chan DurationInvocation, 1000),
	}

	go metricsCollector.metricGatheringLoop()
	go metricsCollector.controlPlaneNotifierLoop()
	go metricsCollector.minuteShiftLoop()

	return metricsCollector.incomingRequestChannel, metricsCollector.doneRequestChannel
}

func (c *MetricsCollector) metricGatheringLoop() {
	for {
		select {
		case <-c.sendChannel:
			c.sendMetricsToControlPlane()
		case <-c.updateChannel:
			c.shitBins()
		case service := <-c.incomingRequestChannel:
			logrus.Tracef("Received data from incomingRequestChannel : %s", service)
			if _, ok := c.metadata[service]; !ok {
				c.metadata[service] = &MetadataPerFunction{
					invocationPerSeconds: make([]uint32, 60),
					averageDuration:      0,
				}
			}

			currentSlice := c.metadata[service]
			currentSlice.invocationPerSeconds[0]++
		case duration := <-c.doneRequestChannel:
			logrus.Tracef("Received data from doneRequestChannel : %s", duration.ServiceName)
			// TODO: Clean this
			tmp := c.metadata[duration.ServiceName]
			tmp.averageDuration = utils.ExponentialMovingAverage(duration.Duration, tmp.averageDuration)
		}
	}
}

func (c *MetricsCollector) controlPlaneNotifierLoop() {
	for {
		// Send values to control plane
		c.sendChannel <- struct{}{}

		// Sleep
		//time.Sleep(time.Minute)
		time.Sleep(time.Duration(c.controlPlaneNotifyInterval) * time.Minute)
	}
}

func (c *MetricsCollector) minuteShiftLoop() {
	for {
		// Trigger refresh
		c.updateChannel <- struct{}{}

		// Sleep
		time.Sleep(time.Minute)
	}
}

func (c *MetricsCollector) sendMetricsToControlPlane() {
	logrus.Infof("Sending values in the control plane")

	// TODO: Change the context
	c.controlPlane.SendMetricsToPredictiveAutoscaler(context.Background(), c.parseMetrics())
}

func (c *MetricsCollector) parseMetrics() *proto.MetricsPredictiveAutoscaler {
	functionNames := make([]string, 0, len(c.metadata))
	functionsDurations := make([]uint32, 0, len(c.metadata))
	invocationsPerMinute := make([]uint32, 0, 60*len(c.metadata))

	for functionName, values := range c.metadata {
		functionNames = append(functionNames, functionName)
		functionsDurations = append(functionsDurations, values.averageDuration)
		invocationsPerMinute = append(invocationsPerMinute, values.invocationPerSeconds...)
	}

	return &proto.MetricsPredictiveAutoscaler{
		FunctionNames:        functionNames,
		FunctionDuration:     functionsDurations,
		InvocationsPerMinute: invocationsPerMinute,
	}
}

func (c *MetricsCollector) shitBins() {
	logrus.Infof("Shifting bins")

	/*for service, functionName := range c.metadata {
		logrus.Infof("Service : %s", service)
		for _, val := range functionName.invocationPerSeconds {
			fmt.Print(fmt.Sprintf("%d | ", val))
		}
		fmt.Println()
		fmt.Println(functionName.averageDuration)
	}*/

	for _, value := range c.metadata {
		for i := number_minutes - 1; i > 0; i-- {
			value.invocationPerSeconds[i] = value.invocationPerSeconds[i-1]
		}
		value.invocationPerSeconds[0] = 0
	}
}
