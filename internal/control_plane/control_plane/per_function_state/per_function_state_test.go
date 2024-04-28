package per_function_state

import (
	"cluster_manager/proto"
	"github.com/stretchr/testify/assert"
	"testing"
	"time"
)

func getScalingChannel() chan int {
	return make(chan int)
}

func getServiceInfo() *proto.ServiceInfo {
	return &proto.ServiceInfo{
		Name:           "mockService",
		Image:          "",
		PortForwarding: nil,
		AutoscalingConfig: &proto.AutoscalingConfiguration{
			ScalingUpperBound:                    1,
			ScalingLowerBound:                    0,
			PanicThresholdPercentage:             0,
			MaxScaleUpRate:                       0,
			MaxScaleDownRate:                     0,
			ContainerConcurrency:                 0,
			ContainerConcurrencyTargetPercentage: 0,
			StableWindowWidthSeconds:             0,
			PanicWindowWidthSeconds:              0,
			ScalingPeriodSeconds:                 0,
		},
	}
}

func TestSimpleController(t *testing.T) {
	scalingChannel := getScalingChannel()
	serviceInfo := getServiceInfo()

	pfStateController := NewPerFunctionState(scalingChannel, serviceInfo, time.Millisecond*100)

	assert.True(t, pfStateController.Start(), "Start should return true")
}

func TestMultipleStarts(t *testing.T) {
	scalingChannel := getScalingChannel()
	serviceInfo := getServiceInfo()

	pfStateController := NewPerFunctionState(scalingChannel, serviceInfo, time.Millisecond*100)
	assert.True(t, pfStateController.Start(), "Start should return true")
	for i := 0; i < 10000; i++ {
		assert.False(t, pfStateController.Start(), "Start should return false")
	}
}
