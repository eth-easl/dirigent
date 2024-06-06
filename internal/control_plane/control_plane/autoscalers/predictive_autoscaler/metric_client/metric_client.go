package metric_client

import (
	"cluster_manager/internal/control_plane/control_plane/function_state"
	"math"
)

type AveragingMethod = int32

const (
	Arithmetic AveragingMethod = iota
	Exponential
)

type MetricClient struct {
	functionState *function_state.FunctionState

	ScalingMetrics []float64
	WindowHead     int64
}

func NewMetricClient(metadata *function_state.FunctionState) *MetricClient {
	return &MetricClient{
		functionState: metadata,
	}
}

func (c *MetricClient) StableAndPanicConcurrency() (float64, float64) {
	observedStableValue := float64(c.functionState.CachedScalingMetrics)

	panicBucketCount := int64(c.functionState.AutoscalingConfig.PanicWindowWidthSeconds / c.functionState.AutoscalingConfig.ScalingPeriodSeconds)
	stableBucketCount := int64(c.functionState.AutoscalingConfig.StableWindowWidthSeconds / c.functionState.AutoscalingConfig.ScalingPeriodSeconds)

	var smoothingCoefficientStable, smoothingCoefficientPanic, multiplierStable, multiplierPanic float64

	if c.functionState.AutoscalingConfig.ScalingMethod == Exponential {
		multiplierStable = smoothingCoefficientStable
		multiplierPanic = smoothingCoefficientPanic
	}

	// because we get metrics every 2s, so 30 buckets are 60s
	if len(c.ScalingMetrics) < int(stableBucketCount) {
		// append value as new bucket if we have not reached max number of buckets
		c.ScalingMetrics = append(c.ScalingMetrics, observedStableValue)
	} else {
		// otherwise replace the least recent measurement
		c.ScalingMetrics[c.WindowHead] = observedStableValue
	}

	currentWindowIndex := c.WindowHead
	avgStable := 0.0

	windowLength := int(math.Min(float64(stableBucketCount), float64(len(c.ScalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		// most recent one has the highest weight
		value := c.ScalingMetrics[currentWindowIndex]
		if c.functionState.AutoscalingConfig.ScalingMethod == Exponential {
			value = value * multiplierStable
			multiplierStable = (1 - smoothingCoefficientStable) * multiplierStable
		}

		avgStable += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	if c.functionState.AutoscalingConfig.ScalingMethod == Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgStable = avgStable / float64(windowLength)
	}

	currentWindowIndex = c.WindowHead
	avgPanic := 0.0

	windowLength = int(math.Min(float64(panicBucketCount), float64(len(c.ScalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		value := c.ScalingMetrics[currentWindowIndex]
		if c.functionState.AutoscalingConfig.ScalingMethod == Exponential {
			value = value * multiplierPanic
			multiplierPanic = (1 - smoothingCoefficientPanic) * multiplierPanic
		}

		avgPanic += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	c.WindowHead++
	if c.WindowHead >= stableBucketCount {
		c.WindowHead = 0 // move windowHead back to 0 if it exceeds the maximum number of buckets
	}

	if c.functionState.AutoscalingConfig.ScalingMethod == Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgPanic = avgPanic / float64(windowLength)
	}

	return avgStable, avgPanic
}

func (c *MetricClient) StableAndPanicRPS() float64 {
	return c.functionState.RpsValue
}
