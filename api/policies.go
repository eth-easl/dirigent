package api

import (
	"cluster_manager/api/proto"
	"math"
	"time"

	"github.com/sirupsen/logrus"
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

	MaxScaleUpRate   float64
	MaxScaleDownRate float64

	ContainerConcurrency                 int
	ContainerConcurrencyTargetPercentage int
	ScalingUpperBound                    int
	ScalingLowerBound                    int

	StableWindowWidth time.Duration
	PanicWindowWidth  time.Duration

	cachedScalingMetric float64
	scalingMetrics      []float64
	windowHead          int64
}

func NewDefaultAutoscalingMetadata() *proto.AutoscalingConfiguration {
	return &proto.AutoscalingConfiguration{
		ScalingUpperBound:                    math.MaxInt32,
		ScalingLowerBound:                    0,
		PanicThresholdPercentage:             10,
		MaxScaleUpRate:                       1000.0,
		MaxScaleDownRate:                     2.0,
		ContainerConcurrency:                 1,
		ContainerConcurrencyTargetPercentage: 70,
		StableWindowWidthSeconds:             6,
		PanicWindowWidthSeconds:              2,
		ScalingPeriodSeconds:                 2,
	}
}

func ConvertProtoToAutoscalingStruct(p *proto.AutoscalingConfiguration) AutoscalingMetadata {
	return AutoscalingMetadata{
		ScalingUpperBound:                    int(p.ScalingUpperBound),
		ScalingLowerBound:                    int(p.ScalingLowerBound),
		PanicThresholdPercentage:             float64(p.PanicThresholdPercentage),
		MaxScaleUpRate:                       float64(p.MaxScaleUpRate),
		MaxScaleDownRate:                     float64(p.MaxScaleDownRate),
		ContainerConcurrency:                 int(p.ContainerConcurrency),
		ContainerConcurrencyTargetPercentage: int(p.ContainerConcurrencyTargetPercentage),
		StableWindowWidth:                    time.Duration(p.StableWindowWidthSeconds) * time.Second,
		PanicWindowWidth:                     time.Duration(p.PanicWindowWidthSeconds) * time.Second,
		ScalingPeriod:                        time.Duration(p.ScalingPeriodSeconds) * time.Second,
	}
}

func (s *AutoscalingMetadata) setCachedScalingMetric(value float64) {
	s.cachedScalingMetric = value
}

func (s *AutoscalingMetadata) KnativeScaling(isScaleFromZero bool) int {
	if isScaleFromZero {
		return 1
	}

	desiredScale, _ := s.internalScaleAlgorithm(s.cachedScalingMetric)

	if desiredScale > s.ScalingUpperBound {
		desiredScale = s.ScalingUpperBound
	} else if desiredScale < s.ScalingLowerBound {
		desiredScale = s.ScalingLowerBound
	}

	return desiredScale
}

func (s *AutoscalingMetadata) internalScaleAlgorithm(scalingMetric float64) (int, float64) {
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
	maxScaleDown := math.Floor(readyPodsCount / s.MaxScaleDownRate)

	desired := float64(0)

	if s.ContainerConcurrency == 0 {
		desired = 100 * (float64(s.ContainerConcurrencyTargetPercentage) / 100)
	} else {
		desired = float64(s.ContainerConcurrency) * (float64(s.ContainerConcurrencyTargetPercentage) / 100)
	}

	var avgStable, avgPanic float64
	avgStable, avgPanic = s.windowAverage(observedStableValue)

	dspc := math.Ceil(avgStable / desired)
	dppc := math.Ceil(avgPanic / desired)

	// We want to keep desired pod count in the  [maxScaleDown, maxScaleUp] range.
	desiredStablePodCount := int(math.Min(math.Max(dspc, maxScaleDown), maxScaleUp))
	desiredPanicPodCount := int(math.Min(math.Max(dppc, maxScaleDown), maxScaleUp))

	isOverPanicThreshold := dppc/readyPodsCount >= (s.PanicThresholdPercentage / 100)

	if !s.InPanicMode && isOverPanicThreshold {
		s.InPanicMode = true
		s.StartPanickingTimestamp = time.Now()

		logrus.Debug("Entered panic mode")
	} else if isOverPanicThreshold {
		// If we're still over panic threshold right now — extend the panic window.
		s.StartPanickingTimestamp = time.Now()

		logrus.Debug("Extended panic mode")
	} else if s.InPanicMode && !isOverPanicThreshold && s.StartPanickingTimestamp.Add(s.StableWindowWidth).Before(time.Now()) {
		// Stop panicking after the surge has made its way into the stable metric.
		s.InPanicMode = false
		s.StartPanickingTimestamp = time.Time{}
		s.MaxPanicPods = 0

		logrus.Debug("Exited panic mode")
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
	return desiredPodCount, avgStable
}

func (s *AutoscalingMetadata) windowAverage(observedStableValue float64) (float64, float64) {
	panicBucketCount := int64(s.PanicWindowWidth / s.ScalingPeriod)
	stableBucketCount := int64(s.StableWindowWidth / s.ScalingPeriod)

	var smoothingCoefficientStable, smoothingCoefficientPanic, multiplierStable, multiplierPanic float64

	if s.ScalingMethod == Exponential {
		multiplierStable = smoothingCoefficientStable
		multiplierPanic = smoothingCoefficientPanic
	}

	// because we get metrics every 2s, so 30 buckets are 60s
	if len(s.scalingMetrics) < int(stableBucketCount) {
		// append value as new bucket if we have not reached max number of buckets
		s.scalingMetrics = append(s.scalingMetrics, observedStableValue)
	} else {
		// otherwise replace the least recent measurement
		s.scalingMetrics[s.windowHead] = observedStableValue
	}

	currentWindowIndex := s.windowHead
	avgStable := 0.0

	windowLength := int(math.Min(float64(stableBucketCount), float64(len(s.scalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		// most recent one has highest weight
		value := s.scalingMetrics[currentWindowIndex]
		if s.ScalingMethod == Exponential {
			value = value * multiplierStable
			multiplierStable = (1 - smoothingCoefficientStable) * multiplierStable
		}

		avgStable += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	if s.ScalingMethod == Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgStable = avgStable / float64(windowLength)
	}

	currentWindowIndex = s.windowHead
	avgPanic := 0.0

	windowLength = int(math.Min(float64(panicBucketCount), float64(len(s.scalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		value := s.scalingMetrics[currentWindowIndex]
		if s.ScalingMethod == Exponential {
			value = value * multiplierPanic
			multiplierPanic = (1 - smoothingCoefficientPanic) * multiplierPanic
		}

		avgPanic += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	s.windowHead++
	if s.windowHead >= stableBucketCount {
		s.windowHead = 0 // move windowHead back to 0 if it exceeds the maximum number of buckets
	}

	if s.ScalingMethod == Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgPanic = avgPanic / float64(windowLength)
	}

	return avgStable, avgPanic
}
