package api

/*import (
	"math"
	"time"
)

type AveragingMethod int64

const (
	Arithmetic AveragingMethod = iota
	Exponential
)

type AutoscalingMetadata struct {
	ActualScale int

	ScalingMethod AveragingMethod
	ScalingPeriod time.Duration

	InPanicMode              bool
	StartPanickingTimestamp  time.Time
	MaxPanicPods             int
	PanicThresholdPercentage float64

	MaxScaleUpRate float64

	ContainerConcurrency                 int
	ContainerConcurrencyTargetPercentage int
	ScalingUpperBound                    int
	ScalingLowerBound                    int

	StableWindowWidth time.Duration
	PanicWindowWidth  time.Duration
}

func NewAutoscalingMetadata() AutoscalingMetadata {
	return AutoscalingMetadata{
		ScalingUpperBound:                    math.MaxInt32,
		ScalingLowerBound:                    0,
		MaxScaleUpRate:                       1000.0,
		ContainerConcurrency:                 1,
		ContainerConcurrencyTargetPercentage: 70,
		StableWindowWidth:                    6 * time.Second,
		PanicWindowWidth:                     2 * time.Second,
	}
}

func (s *AutoscalingMetadata) knativeScaling(functionHash string, scalingMetric float64) (int, int, float64) {
	desiredScale, avgScalingMetric := s.kNativeScaleAlgorithm(functionHash, scalingMetric)

	if desiredScale > s.ScalingUpperBound {
		desiredScale = s.ScalingUpperBound
	} else if desiredScale < s.ScalingLowerBound {
		desiredScale = s.ScalingLowerBound
	}

	return s.ActualScale, desiredScale, avgScalingMetric
}

func (s *AutoscalingMetadata) kNativeScaleAlgorithm(functionHash string, scalingMetric float64) (int, float64) {
	originalReadyPodsCount := s.ActualScale

	// Use 1 if there are zero current pods.
	readyPodsCount := math.Max(1, float64(originalReadyPodsCount))

	// concurrency is used by default
	var observedStableValue = scalingMetric

	// Make sure we don't get stuck with the same number of pods, if the scale up rate
	// is too conservative and MaxScaleUp*RPC==RPC, so this permits us to grow at least by a single
	// pod if we need to scale up.
	// E.g. MSUR=1.1, OCC=3, RPC=2, TV=1 => OCC/TV=3, MSU=2.2 => DSPC=2, while we definitely, need
	// 3 pods. See the unit test for this scenario in action.
	maxScaleUp := math.Ceil(s.MaxScaleUpRate * readyPodsCount)
	// Same logic, opposite math applies here.
	maxScaleDown := 0.

	desired := float64(0)

	if s.ContainerConcurrency == 0 {
		desired = 100 * (float64(s.ContainerConcurrencyTargetPercentage) / 100)
	} else {
		desired = float64(s.ContainerConcurrency) * (float64(s.ContainerConcurrencyTargetPercentage) / 100)
	}

	var avgStable, avgPanic float64
	avgStable, avgPanic = s.windowAverage(functionHash, observedStableValue)

	dspc := math.Ceil(avgStable / desired)
	dppc := math.Ceil(avgPanic / desired)

	// We want to keep desired pod count in the  [maxScaleDown, maxScaleUp] range.
	desiredStablePodCount := int(math.Min(math.Max(dspc, maxScaleDown), maxScaleUp))
	desiredPanicPodCount := int(math.Min(math.Max(dppc, maxScaleDown), maxScaleUp))

	isOverPanicThreshold := dppc/readyPodsCount >= (s.PanicThresholdPercentage / 100)

	if !s.InPanicMode && isOverPanicThreshold {
		s.InPanicMode = true
		s.StartPanickingTimestamp = time.Now()
	} else if isOverPanicThreshold {
		// If we're still over panic threshold right now — extend the panic window.
		s.StartPanickingTimestamp = time.Now()
	} else if s.InPanicMode && !isOverPanicThreshold && time.Since(s.StartPanickingTimestamp) > s.StableWindowWidth {
		// Stop panicking after the surge has made its way into the stable metric.
		s.InPanicMode = false
		s.StartPanickingTimestamp = time.Time{}
		s.MaxPanicPods = 0
	}

	desiredPodCount := desiredStablePodCount
	if s.InPanicMode {
		// In some edge cases stable window metric might be larger
		// than panic one. And we should provision for stable as for panic,
		// so pick the larger of the two.
		if desiredPodCount < desiredPanicPodCount {
			desiredPodCount = desiredPanicPodCount
		}
		//logger.Debug("Operating in panic mode.")
		// We do not scale down while in panic mode. Only increases will be applied.
		if desiredPodCount > s.MaxPanicPods {
			//	logger.Infof("Increasing pods count from %d to %d.", originalReadyPodsCount, desiredPodCount)
			s.MaxPanicPods = desiredPodCount
		}

		desiredPodCount = s.MaxPanicPods
	}
	/*if autoscalingAlgorithm == abstractions.AS_KNATIVE_RPS_MODE {
		if desiredPodCount == 0 && s.requestsInQueue[functionHash] > 0 {
			desiredPodCount = 1
			// only allow scaling down to 0 if queue is empty
		}
	}*/
/*return desiredPodCount, avgStable
}

func (s *AutoscalingMetadata) windowAverage(functionHash string, observedStableValue float64) (float64, float64) {
	panicBucketCount := s.PanicWindowWidth / s.ScalingPeriod
	stableBucketCount := s.StableWindowWidth / s.ScalingPeriod

	var smoothingCoefficientStable, smoothingCoefficientPanic, multiplierStable, multiplierPanic float64

	if s.ScalingMethod == Exponential {
		multiplierStable = smoothingCoefficientStable
		multiplierPanic = smoothingCoefficientPanic
	}

	// because we get metrics every 2s, so 30 buckets are 60s
	if len(s.scalingMetrics[functionHash]) < int(stableBucketCount) {
		// append value as new bucket if we have not reached max number of buckets
		s.scalingMetrics[functionHash] = append(s.scalingMetrics[functionHash], observedStableValue)
	} else {
		// otherwise replace the least recent measurement
		s.scalingMetrics[functionHash][s.windowHead[functionHash]] = observedStableValue
	}

	currentWindowIndex := s.windowHead[functionHash]
	avgStable := 0.0

	windowLength := int(math.Min(float64(stableBucketCount), float64(len(s.scalingMetrics[functionHash]))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		// most recent one has highest weight
		value := s.scalingMetrics[functionHash][currentWindowIndex]
		if s.aggregationMethod[functionHash] == abstractions.AGGREGATION_EXPONENTIAL {
			value = value * multiplierStable
			multiplierStable = (1 - smoothingCoefficientStable) * multiplierStable
		}
		avgStable += value
		currentWindowIndex--
		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	if s.aggregationMethod[functionHash] == abstractions.AGGREGATION_ARITHMETIC {
		avgStable = avgStable / float64(windowLength)
		// divide by the number of buckets we summed over to get the average
	}

	currentWindowIndex = s.windowHead[functionHash]
	avgPanic := 0.0

	windowLength = int(math.Min(float64(panicBucketCount), float64(len(s.scalingMetrics[functionHash]))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		value := s.scalingMetrics[functionHash][currentWindowIndex]
		if s.aggregationMethod[functionHash] == abstractions.AGGREGATION_EXPONENTIAL {
			value = value * multiplierPanic
			multiplierPanic = (1 - smoothingCoefficientPanic) * multiplierPanic
		}
		avgPanic += value
		currentWindowIndex--
		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	s.windowHead[functionHash]++
	if s.windowHead[functionHash] >= stableBucketCount {
		s.windowHead[functionHash] = 0 // move windowHead back to 0 if it exceeds the maximum number of buckets
	}

	if s.aggregationMethod[functionHash] == abstractions.AGGREGATION_ARITHMETIC {
		avgPanic = avgPanic / float64(windowLength)
		// divide by the number of buckets we summed over to get the average
	}

	return avgStable, avgPanic
}*/
