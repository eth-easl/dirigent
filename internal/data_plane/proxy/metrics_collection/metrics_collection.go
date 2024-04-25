package metrics_collection

import (
	"cluster_manager/pkg/utils"
	"github.com/sirupsen/logrus"
	"time"
)

const number_minutes int = 60

type DurationInvocation struct {
	Duration    uint32
	ServiceName string
}

type MetricsCollector struct {
	current         [number_minutes]map[string]uint
	averageDuration map[string]uint32

	controlPlaneNotifyInterval int

	sendChannel   chan struct{}
	updateChannel chan struct{}

	incomingRequestChannel chan string
	doneRequestChannel     chan DurationInvocation
}

func NewMetricsCollector(controlPlaneNotifyInterval int) (chan string, chan DurationInvocation) {
	metricsCollector := &MetricsCollector{
		current:         [number_minutes]map[string]uint(make([]map[string]uint, number_minutes)),
		averageDuration: make(map[string]uint32),

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
			c.current[0][service]++
		case duration := <-c.doneRequestChannel:
			logrus.Tracef("Received data from doneRequestChannel : %s", duration.ServiceName)
			c.averageDuration[duration.ServiceName] = utils.ExponentialMovingAverage(duration.Duration, c.averageDuration[duration.ServiceName])
		}
	}
}

func (c *MetricsCollector) controlPlaneNotifierLoop() {
	for {
		// Send values to control plane
		c.sendChannel <- struct{}{}

		// Sleep
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

	// TODO: Implement the gRPC call as second milestone
}

func (c *MetricsCollector) shitBins() {
	logrus.Infof("Shifting bins")

	for i := number_minutes - 1; i > 0; i-- {
		c.current[i] = c.current[i-1]
	}

	c.current[0] = make(map[string]uint)
}
