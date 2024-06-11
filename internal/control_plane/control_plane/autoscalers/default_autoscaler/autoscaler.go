package default_autoscaler

import (
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/internal/control_plane/control_plane/function_state"
	"github.com/sirupsen/logrus"
	"math"
	"sync/atomic"
	"time"

	"github.com/cznic/mathutil"
)

type defaultAutoscaler struct {
	InPanicMode             bool
	StartPanickingTimestamp time.Time
	MaxPanicPods            int

	ScalingMetrics []float64
	WindowHead     int64

	AutoscalingRunning int32

	Period time.Duration

	functionState *function_state.FunctionState
}

func newDefaultAutoscaler(functionState *function_state.FunctionState, autoscalingPeriod time.Duration) *defaultAutoscaler {
	return &defaultAutoscaler{
		Period:        autoscalingPeriod,
		functionState: functionState,
	}
}

func (s *defaultAutoscaler) panicPoke() {
	s.InPanicMode = true
	s.StartPanickingTimestamp = time.Now()
	atomic.AddInt64(&s.functionState.ActualScale, 1)

	s.poke()
}

func (s *defaultAutoscaler) poke() {
	if atomic.CompareAndSwapInt32(&s.AutoscalingRunning, 0, 1) {
		logrus.Warn(s.functionState)
		s.functionState.StopCh = make(chan struct{})
		go s.scalingLoop()
	}
}

func (s *defaultAutoscaler) stop() {
	if atomic.LoadInt32(&s.AutoscalingRunning) == 1 {
		s.functionState.StopCh <- struct{}{}
	}
}

func (s *defaultAutoscaler) scalingCycle(isScaleFromZero bool) (stopped bool) {
	desiredScale := s.KnativeScaling(isScaleFromZero)
	logrus.Debugf("Desired scale for %s is %d", s.functionState.ServiceName, desiredScale)

	s.functionState.DesiredStateChannel <- desiredScale

	if desiredScale == 0 {
		s.stopAutoscalingLoop()
		return true
	}

	return false
}

func (s *defaultAutoscaler) stopAutoscalingLoop() {
	logrus.Debugf("Exited scaling loop for %s.", s.functionState.ServiceName)

	atomic.StoreInt32(&s.AutoscalingRunning, 0)

	// TODO: This code sometimes crashes
	/* close(s.functionState.StopCh)
	s.functionState.StopCh = nil */
}

func (s *defaultAutoscaler) scalingLoop() {
	logrus.Debugf("Starting scaling loop for %s.", s.functionState.ServiceName)

	// need to make the first tick happen right away
	toStop := s.scalingCycle(true)
	if toStop {
		return
	}

	ticker := time.NewTicker(s.Period)
	for {
		select {
		case <-ticker.C: // first event only after as.Period delay
			toStop = s.scalingCycle(false)
			if toStop {
				return
			}
		case <-s.functionState.StopCh:
			s.stopAutoscalingLoop()
			return
		}
	}
}

func (s *defaultAutoscaler) KnativeScaling(isScaleFromZero bool) int {
	if isScaleFromZero {
		return 1
	}

	desiredScale, _ := s.internalScaleAlgorithm(float64(s.functionState.CachedScalingMetrics))

	return mathutil.Clamp(desiredScale, int(s.functionState.AutoscalingConfig.ScalingLowerBound), int(s.functionState.AutoscalingConfig.ScalingUpperBound))
}

func (s *defaultAutoscaler) internalScaleAlgorithm(scalingMetric float64) (int, float64) {
	originalReadyPodsCount := s.functionState.ActualScale

	// Use 1 if there are zero current pods.
	readyPodsCount := math.Max(1, float64(originalReadyPodsCount))

	// concurrency is used by default
	var observedStableValue = scalingMetric

	// Make sure we don't get stuck with the same number of pods, if the scale up rate
	// is too conservative and MaxScaleUp*RPC==RPC, so this permits us to grow at least by a single
	// pod if we need to scale up.
	// E.g. MSUR=1.1, OCC=3, RPC=2, TV=1 => OCC/TV=3, MSU=2.2 => DSPC=2, while we definitely, need
	// 3 pods. See the unit test for this scenario in action.
	maxScaleUp := math.Ceil(float64(s.functionState.AutoscalingConfig.MaxScaleUpRate) * readyPodsCount)
	// Same logic, opposite math applies here.
	maxScaleDown := math.Floor(readyPodsCount / float64(s.functionState.AutoscalingConfig.MaxScaleDownRate))

	desired := float64(0)

	if s.functionState.AutoscalingConfig.ContainerConcurrency == 0 {
		desired = 100 * (float64(s.functionState.AutoscalingConfig.ContainerConcurrencyTargetPercentage) / 100)
	} else {
		desired = float64(s.functionState.AutoscalingConfig.ContainerConcurrency) * (float64(s.functionState.AutoscalingConfig.ContainerConcurrencyTargetPercentage) / 100)
	}

	var avgStable, avgPanic float64
	avgStable, avgPanic = s.windowAverage(observedStableValue)

	dspc := math.Ceil(avgStable / desired)
	dppc := math.Ceil(avgPanic / desired)

	// We want to keep desired pod count in the  [maxScaleDown, maxScaleUp] range.
	desiredStablePodCount := int(math.Min(math.Max(dspc, maxScaleDown), maxScaleUp))
	desiredPanicPodCount := int(math.Min(math.Max(dppc, maxScaleDown), maxScaleUp))

	isOverPanicThreshold := dppc/readyPodsCount >= (float64(s.functionState.AutoscalingConfig.PanicThresholdPercentage) / 100)

	if !s.InPanicMode && isOverPanicThreshold {
		s.InPanicMode = true
		s.StartPanickingTimestamp = time.Now()

		logrus.Debug("Entered panic mode")
	} else if isOverPanicThreshold {
		// If we're still over panic threshold right now — extend the panic window.
		s.StartPanickingTimestamp = time.Now()

		logrus.Debug("Extended panic mode")
	} else if s.InPanicMode && !isOverPanicThreshold &&
		s.StartPanickingTimestamp.Add(time.Duration(s.functionState.AutoscalingConfig.StableWindowWidthSeconds)*time.Second).Before(time.Now()) {
		// stop panicking after the surge has made its way into the stable metric.
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

func (s *defaultAutoscaler) windowAverage(observedStableValue float64) (float64, float64) {
	panicBucketCount := int64(s.functionState.AutoscalingConfig.PanicWindowWidthSeconds / s.functionState.AutoscalingConfig.ScalingPeriodSeconds)
	stableBucketCount := int64(s.functionState.AutoscalingConfig.StableWindowWidthSeconds / s.functionState.AutoscalingConfig.ScalingPeriodSeconds)

	var smoothingCoefficientStable, smoothingCoefficientPanic, multiplierStable, multiplierPanic float64

	if s.functionState.AutoscalingConfig.ScalingMethod == core.Exponential {
		multiplierStable = smoothingCoefficientStable
		multiplierPanic = smoothingCoefficientPanic
	}

	// because we get metrics every 2s, so 30 buckets are 60s
	if len(s.ScalingMetrics) < int(stableBucketCount) {
		// append value as new bucket if we have not reached max number of buckets
		s.ScalingMetrics = append(s.ScalingMetrics, observedStableValue)
	} else {
		// otherwise replace the least recent measurement
		s.ScalingMetrics[s.WindowHead] = observedStableValue
	}

	currentWindowIndex := s.WindowHead
	avgStable := 0.0

	windowLength := int(math.Min(float64(stableBucketCount), float64(len(s.ScalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		// most recent one has the highest weight
		value := s.ScalingMetrics[currentWindowIndex]
		if s.functionState.AutoscalingConfig.ScalingMethod == core.Exponential {
			value = value * multiplierStable
			multiplierStable = (1 - smoothingCoefficientStable) * multiplierStable
		}

		avgStable += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	if s.functionState.AutoscalingConfig.ScalingMethod == core.Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgStable = avgStable / float64(windowLength)
	}

	currentWindowIndex = s.WindowHead
	avgPanic := 0.0

	windowLength = int(math.Min(float64(panicBucketCount), float64(len(s.ScalingMetrics))))
	for i := 0; i < windowLength; i++ {
		// sum values of buckets, starting at most recent measurement
		value := s.ScalingMetrics[currentWindowIndex]
		if s.functionState.AutoscalingConfig.ScalingMethod == core.Exponential {
			value = value * multiplierPanic
			multiplierPanic = (1 - smoothingCoefficientPanic) * multiplierPanic
		}

		avgPanic += value
		currentWindowIndex--

		if currentWindowIndex < 0 {
			currentWindowIndex = int64(windowLength) - 1
		}
	}

	s.WindowHead++
	if s.WindowHead >= stableBucketCount {
		s.WindowHead = 0 // move WindowHead back to 0 if it exceeds the maximum number of buckets
	}

	if s.functionState.AutoscalingConfig.ScalingMethod == core.Arithmetic {
		// divide by the number of buckets we summed over to get the average
		avgPanic = avgPanic / float64(windowLength)
	}

	return avgStable, avgPanic
}
