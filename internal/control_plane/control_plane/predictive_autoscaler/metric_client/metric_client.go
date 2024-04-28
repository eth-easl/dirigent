package metric_client

import (
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"math"
)

type AveragingMethod = int32

const (
	Arithmetic AveragingMethod = iota
	Exponential
)

type MetricClient struct {
	autoscalingMetadata *per_function_state.AutoscalingMetadata
}

func NewMetricClient(metadata *per_function_state.AutoscalingMetadata) *MetricClient {
	return &MetricClient{
		autoscalingMetadata: metadata,
	}
}

// WE can simply ask like that as we call every two seconds
func (c *MetricClient) StableAndPanicConcurrency() (float64, float64) {
	// TODO: Change type here
	observedStableValue := float64(c.autoscalingMetadata.CachedScalingMetrics)
	panicBucketCount := int64(c.autoscalingMetadata.AutoscalingConfig.PanicWindowWidthSeconds / c.autoscalingMetadata.AutoscalingConfig.ScalingPeriodSeconds)
	stableBucketCount := int64(c.autoscalingMetadata.AutoscalingConfig.StableWindowWidthSeconds / c.autoscalingMetadata.AutoscalingConfig.ScalingPeriodSeconds)

	var smoothingCoefficientStable, smoothingCoefficientPanic, multiplierStable, multiplierPanic float64

	if c.autoscalingMetadata.AutoscalingConfig.ScalingMethod == Exponential {
		multiplierStable = smoothingCoefficientStable
		multiplierPanic = smoothingCoefficientPanic
	}

	// because we get metrics every 2s, so 30 buckets are 60s
	if len(c.autoscalingMetadata.ScalingMetrics) < int(stableBucketCount) {
		// append value as new bucket if we have not reached max number of buckets
		c.autoscalingMetadata.ScalingMetrics = append(c.autoscalingMetadata.ScalingMetrics, observedStableValue)
	} else {
		// otherwise replace the least recent measurement
		c.autoscalingMetadata.ScalingMetrics[c.autoscalingMetadata.WindowHead] = observedStableValue
	}

	currentWindowIndex := c.autoscalingMetadata.WindowHead
	avgStable := 0.0

	windowLength := int(math.Min(float64(stableBucketCount), float64(len(c.autoscalingMetadata.ScalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		// most recent one has the highest weight
		value := c.autoscalingMetadata.ScalingMetrics[currentWindowIndex]
		if c.autoscalingMetadata.AutoscalingConfig.ScalingMethod == Exponential {
			value = value * multiplierStable
			multiplierStable = (1 - smoothingCoefficientStable) * multiplierStable
		}

		avgStable += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	if c.autoscalingMetadata.AutoscalingConfig.ScalingMethod == Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgStable = avgStable / float64(windowLength)
	}

	currentWindowIndex = c.autoscalingMetadata.WindowHead
	avgPanic := 0.0

	windowLength = int(math.Min(float64(panicBucketCount), float64(len(c.autoscalingMetadata.ScalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		value := c.autoscalingMetadata.ScalingMetrics[currentWindowIndex]
		if c.autoscalingMetadata.AutoscalingConfig.ScalingMethod == Exponential {
			value = value * multiplierPanic
			multiplierPanic = (1 - smoothingCoefficientPanic) * multiplierPanic
		}

		avgPanic += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	c.autoscalingMetadata.WindowHead++
	if c.autoscalingMetadata.WindowHead >= stableBucketCount {
		c.autoscalingMetadata.WindowHead = 0 // move windowHead back to 0 if it exceeds the maximum number of buckets
	}

	if c.autoscalingMetadata.AutoscalingConfig.ScalingMethod == Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgPanic = avgPanic / float64(windowLength)
	}

	return avgStable, avgPanic
}
