package predictive_autoscaler

import (
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"cluster_manager/internal/control_plane/control_plane/predictive_autoscaler/metric_client"
	"github.com/sirupsen/logrus"
	"math"
	"time"

	"knative.dev/serving/pkg/autoscaler/aggregation/max"
)

// autoscaler stores current state of an instance of an autoscaler.
type autoscaler struct {
	namespace string
	revision  string
	// TODO: Integrate it
	metricClient  *metric_client.MetricClient
	functionState *per_function_state.PFState
	// State in panic mode.
	panicTime    time.Time
	maxPanicPods int32

	// delayWindow is used to defer scale-down decisions until a time
	// window has passed at the reduced concurrency.
	delayWindow *max.TimeWindow

	// predictive per_function_state

	predictionsCh    chan ScalingDecisions
	shiftedScalingCh chan ScalingDecisions
	startCh          chan bool

	startPredictiveScaling bool

	invocationsPerMinute []float64
	startTime            time.Time
	currentMinute        int
	currentEpoch         int
	averageCapacity      float64
	predictedScale       []int32
	shiftedScale         []int32
	limitsComputed       bool
	prevPredictionMinute int

	// oracle
	scale        []int
	epochCounter int
	timeToWait   int64

	// Mu policy

	// custom scalar parameters
	maxWaitingTimeInSec   float64
	executionTimeInSec    float64
	incomingRequestPerSec float64
	totalServingRate      float64
	maxPerPodServingRate  float64
	prevDesiredPodCount   float64
	DecreaseCounter       int32

	// for CalculatePodCountUsingCustomSLO()
	graceFlag                 int
	desiredPodCountSLO_old    float64
	incomingRequestPerSec_old float64
	totalServingRate_old      float64
	PerPodServingRate_old     float64
	servingRate_list          []float64
	servingRate_list_index    int32

	podCPUUsage_old float64
	podMemUsage_old float64

	// Linear Predictor
	modelList []ModelUnit
	pattern   Pattern
	MaxIRSeen float64

	// For Dirigent only and mu
	rps float64
}

// New creates a new instance of default autoscaler implementation.
func New(
	functionState *per_function_state.PFState,
	revision string,
	deciderSpec *DeciderSpec,
	predictionsCh chan ScalingDecisions,
	shiftedScalingCh chan ScalingDecisions,
	startCh chan bool) UniScaler {

	var delayer *max.TimeWindow
	if deciderSpec.ScaleDownDelay > 0 {
		delayer = max.NewTimeWindow(deciderSpec.ScaleDownDelay, tickInterval)
	}

	return newAutoscaler(functionState, revision, delayer, predictionsCh, shiftedScalingCh, startCh)
}

func newAutoscaler(
	functionState *per_function_state.PFState,
	revision string,
	delayWindow *max.TimeWindow,
	predictionsCh chan ScalingDecisions,
	shiftedScalingCh chan ScalingDecisions,
	startCh chan bool) *autoscaler {

	// We always start in the panic mode, if the deployment is scaled up over 1 pod.
	// If the scale is 0 or 1, normal Autoscaler behavior is fine.
	// When Autoscaler restarts we lose metric history, which causes us to
	// momentarily scale down, and that is not a desired behaviour.
	// Thus, we're keeping at least the current scale until we
	// accumulate enough data to make conscious decisions.
	curC := functionState.GetNumberEndpoint()
	var pt time.Time
	if curC > 1 {
		pt = time.Now()
		// A new instance of autoscaler is created in panic mode.
	}

	servingRate_list := make([]float64, 10)

	//List of models with different learning rates
	lrate := 0.1
	var ModelList []ModelUnit

	for lrate > 0.0000000001 {
		//ModelList = append(ModelList, ModelUnit{Weights: make([]float64, windowSize), Bias: 0, Lrate: lrate, WindowSize: windowSize})
		ModelList = append(ModelList, ModelUnit{Weights: make([]float64, 10), Bias: 0, Lrate: lrate, WindowSize: 10})
		ModelList = append(ModelList, ModelUnit{Weights: make([]float64, 50), Bias: 0, Lrate: lrate, WindowSize: 50})
		ModelList = append(ModelList, ModelUnit{Weights: make([]float64, 100), Bias: 0, Lrate: lrate, WindowSize: 100})
		ModelList = append(ModelList, ModelUnit{Weights: make([]float64, 500), Bias: 0, Lrate: lrate, WindowSize: 500})
		ModelList = append(ModelList, ModelUnit{Weights: make([]float64, 1000), Bias: 0, Lrate: lrate, WindowSize: 1000})
		lrate *= 0.1
	}

	return &autoscaler{
		functionState: functionState,
		revision:      revision,

		metricClient:         metric_client.NewMetricClient(functionState),
		invocationsPerMinute: make([]float64, 60),

		delayWindow: delayWindow,

		panicTime:    pt,
		maxPanicPods: int32(curC),

		predictionsCh:    predictionsCh,
		shiftedScalingCh: shiftedScalingCh,
		startCh:          startCh,

		servingRate_list:       servingRate_list,
		servingRate_list_index: 0,
		modelList:              ModelList,
		pattern:                Pattern{Features: make([]float64, 1000), SingleExpectation: 0},
	}
}

// Scale calculates the desired scale based on current statistics given the current time.
// desiredPodCount is the calculated pod count the autoscaler would like to set.
// validScale signifies whether the desiredPodCount should be applied or not.
// Scale is not thread safe in regards to panic state, but it's thread safe in
// regards to acquiring the decider spec.
func (a *autoscaler) Scale(now time.Time) ScaleResult {
	originalReadyPodsCount := a.functionState.GetNumberEndpoint()
	// Use 1 if there are zero current pods.
	readyPodsCount := math.Max(1, float64(originalReadyPodsCount))

	var dspc float64

	metricName := spec.ScalingMetric
	var observedStableValue, observedPanicValue float64
	var dspc, dppc float64

	switch spec.ScalingMetric {
	case utils.PREDICTIVE_AUTOSCALER:
		dspc = a.predictiveAutoscaling(readyPodsCount, now)
		if dspc == -1 {
			return invalidSR
		}
		var excessBCF float64 = -1
		return ScaleResult{
			DesiredPodCount:     int32(dspc),
			ExcessBurstCapacity: int32(excessBCF),
			ScaleValid:          true,
		}
	case utils.MU_POLICY:
		dspc = float64(a.muAutoscaling(readyPodsCount, metricKey, now))
		if dspc == -1 {
			return invalidSR
		}
	case utils.ORACLE_AUTOSCALER:
		dspc = a.oracleScaling(readyPodsCount, metricKey, now)
	default:
		panic("We don't a valid scaling metric")
	}

	// Make sure we don't get stuck with the same number of pods, if the scale up rate
	// is too conservative and MaxScaleUp*RPC==RPC, so this permits us to grow at least by a single
	// pod if we need to scale up.
	// E.g. MSUR=1.1, OCC=3, RPC=2, TV=1 => OCC/TV=3, MSU=2.2 => DSPC=2, while we definitely, need
	// 3 pods. See the unit test for this scenario in action.
	maxScaleUp := math.Ceil(spec.MaxScaleUpRate * readyPodsCount)
	// Same logic, opposite math applies here.
	maxScaleDown := 0.
	if spec.Reachable {
		maxScaleDown = math.Floor(readyPodsCount / spec.MaxScaleDownRate)
	}

	if spec.ScalingMetric == utils.CONCURRENCY_SCALING {
		dspc = math.Ceil(observedStableValue / spec.TargetValue)
		dppc = math.Ceil(observedPanicValue / spec.TargetValue)
		if debugEnabled {
			logrus.Debug(
				fmt.Sprintf("For metric %s observed values: stable = %0.3f; panic = %0.3f; target = %0.3f "+
					"Desired StablePodCount = %0.0f, PanicPodCount = %0.0f, ReadyEndpointCount = %d, MaxScaleUp = %0.0f, MaxScaleDown = %0.0f",
					metricName, observedStableValue, observedPanicValue, spec.TargetValue,
					dspc, dppc, originalReadyPodsCount, maxScaleUp, maxScaleDown))
		}
	}

	// We want to keep desired pod count in the  [maxScaleDown, maxScaleUp] range.
	desiredStablePodCount := int32(math.Min(math.Max(dspc, maxScaleDown), maxScaleUp))
	desiredPanicPodCount := int32(math.Min(math.Max(dppc, maxScaleDown), maxScaleUp))

	var excessBCF float64
	desiredPodCount := desiredStablePodCount
	//	If ActivationScale > 1, then adjust the desired pod counts
	if spec.ScalingMetric == utils.CONCURRENCY_SCALING {
		if a.deciderSpec.ActivationScale > 1 {
			if dspc > 0 && a.deciderSpec.ActivationScale > desiredStablePodCount {
				desiredStablePodCount = a.deciderSpec.ActivationScale
			}
			if dppc > 0 && a.deciderSpec.ActivationScale > desiredPanicPodCount {
				desiredPanicPodCount = a.deciderSpec.ActivationScale
			}
		}

		isOverPanicThreshold := dppc/readyPodsCount >= spec.PanicThreshold

		if a.panicTime.IsZero() && isOverPanicThreshold {
			// Begin panicking when we cross the threshold in the panic window.
			logrus.Info("PANICKING.")
			a.panicTime = now
		} else if isOverPanicThreshold {
			// If we're still over panic threshold right now — extend the panic window.
			a.panicTime = now
		} else if !a.panicTime.IsZero() && !isOverPanicThreshold && a.panicTime.Add(spec.StableWindow).Before(now) {
			// Stop panicking after the surge has made its way into the stable metric.
			logrus.Info("Un-panicking.")
			a.panicTime = time.Time{}
			a.maxPanicPods = 0
		}

		if !a.panicTime.IsZero() {
			// In some edgecases stable window metric might be larger
			// than panic one. And we should provision for stable as for panic,
			// so pick the larger of the two.
			if desiredPodCount < desiredPanicPodCount {
				desiredPodCount = desiredPanicPodCount
			}
			logrus.Debug("Operating in panic mode.")
			// We do not scale down while in panic mode. Only increases will be applied.
			if desiredPodCount > a.maxPanicPods {
				logrus.Infof("Increasing pods count from %d to %d.", originalReadyPodsCount, desiredPodCount)
				a.maxPanicPods = desiredPodCount
			} else if desiredPodCount < a.maxPanicPods {
				logrus.Infof("Skipping pod count decrease from %d to %d.", a.maxPanicPods, desiredPodCount)
			}
			desiredPodCount = a.maxPanicPods
		} else {
			logrus.Debug("Operating in stable mode.")
		}

		// Delay scale down decisions, if a ScaleDownDelay was specified.
		// We only do this if there's a non-nil delayWindow because although a
		// one-element delay window is _almost_ the same as no delay at all, it is
		// not the same in the case where two Scale()s happen in the same time
		// interval (because the largest will be picked rather than the most recent
		// in that case).
		if a.delayWindow != nil {
			a.delayWindow.Record(now, desiredPodCount)
			delayedPodCount := a.delayWindow.Current()
			if delayedPodCount != desiredPodCount {
				if debugEnabled {
					logrus.Debug(
						fmt.Sprintf("Delaying scale to %d, staying at %d",
							desiredPodCount, delayedPodCount))
				}
				desiredPodCount = delayedPodCount
			}
		}

		// Compute excess burst capacity
		//
		// the excess burst capacity is based on panic value, since we don't want to
		// be making knee-jerk decisions about Activator in the request path.
		// Negative EBC means that the deployment does not have enough capacity to serve
		// the desired burst off hand.
		// EBC = TotCapacity - Cur#ReqInFlight - TargetBurstCapacity
		excessBCF = -1.
		switch {
		case spec.TargetBurstCapacity == 0:
			excessBCF = 0
		case spec.TargetBurstCapacity > 0:
			totCap := float64(originalReadyPodsCount) * spec.TotalValue
			excessBCF = math.Floor(totCap - spec.TargetBurstCapacity - observedPanicValue)
		}
	}

	if debugEnabled {
		logrus.Debug(fmt.Sprintf("PodCount=%d Total1PodCapacity=%0.3f ObsStableValue=%0.3f ObsPanicValue=%0.3f TargetBC=%0.3f ExcessBC=%0.3f",
			originalReadyPodsCount, spec.TotalValue, observedStableValue,
			observedPanicValue, spec.TargetBurstCapacity, excessBCF))
	}

	return ScaleResult{
		DesiredPodCount:     desiredPodCount,
		ExcessBurstCapacity: int32(excessBCF),
		ScaleValid:          true,
	}
}

func (a *autoscaler) GetDesiredStateChannel() chan int {
	return a.functionState.DesiredStateChannel
}

func (a *autoscaler) SharePredictions() {
	pred := make([]int32, len(a.predictedScale))
	copy(pred, a.predictedScale)
	s := ScalingDecisions{scale: pred}
	a.predictionsCh <- s
}

func (a *autoscaler) LimitScalingDecisions() ([]int32, bool) {
	var l int32
	newLimit := false
	var limit []int32
	for l != -1 {
		select {
		case s := <-a.shiftedScalingCh:
			limit = s.scale
			newLimit = true
		default:
			l = -1
		}
	}
	return limit, newLimit
}

func (a *autoscaler) predictiveAutoscaling(readyPodsCount float64,
	now time.Time) float64 {
	if !a.startPredictiveScaling {
		logrus.Tracef("Autoscaler %s checking if it can start", a.revision)
		select {
		case _ = <-a.startCh:
			a.startPredictiveScaling = true
			logrus.Tracef("Autoscaler %s starting", a.revision)
		default:
			a.startPredictiveScaling = false
			observedConcurrency, observedPanic := a.metricClient.StableAndPanicConcurrency()
			logrus.Tracef("Autoscaler %s not started returning %f, panic %f",
				a.revision, observedConcurrency, observedPanic)
			return math.Ceil(observedConcurrency)
		}
	}

	if a.startTime.IsZero() {
		a.startTime = now
		a.currentMinute = 0
		a.currentEpoch = 0
	}
	prevMinute := a.currentMinute
	a.currentMinute = int(now.Sub(a.startTime).Minutes())
	logrus.Infof("prevMinute: %d currentMinute: %d currentMinuteAsFloat: %f",
		prevMinute, a.currentMinute, now.Sub(a.startTime).Minutes())

	observedConcurrency, _ := a.metricClient.StableAndPanicConcurrency()

	if a.currentMinute < 60 {
		// purely concurrency based scaling for first 60 minutes
		logrus.Infof("Autoscaler %s using concurrency-based scaling at minute %d with concurrency %f",
			a.revision, a.currentMinute, observedConcurrency)
		a.currentEpoch++
		return math.Ceil(observedConcurrency)
	}

	// This is only executed after 60 minutes of gathering data!
	var prediction []float64
	if a.currentMinute > a.prevPredictionMinute+1 {
		a.prevPredictionMinute = a.currentMinute
		invocationsWindow := a.invocationsPerMinute[len(a.invocationsPerMinute)-60 : len(a.invocationsPerMinute)]
		// we only use fft on a window of the past 60 invocations per minute
		prediction = fourierExtrapolation(invocationsWindow, 10, 60)
		scale := a.predictionToScale(prediction)
		pred := append([]int32{int32(readyPodsCount)}, scale...)
		a.predictedScale = pred
		a.SharePredictions()
		logrus.Infof("Autoscaler %s sharing predictions of length %d at epoch %d",
			a.revision, len(pred), a.currentEpoch)
		logrus.Info("Autoscaler "+a.revision+" sharing predictedScale as ", a.predictedScale)
		s := make([]int32, a.currentEpoch-1)
		s = append(s, pred...)
		a.predictedScale = s
	}

	limit, newLimit := a.LimitScalingDecisions()
	if newLimit {
		a.limitsComputed = true
		scale := make([]int32, a.currentEpoch-1)
		scale = append(scale, limit...)
		a.shiftedScale = scale
		logrus.Infof("Autoscaler %s received limits of length %d at current epoch %d",
			a.revision, len(limit), a.currentEpoch)
		logrus.Info("Autoscaler "+a.revision+"set shiftedScale as ", a.shiftedScale)
	}

	desiredScale := a.predictedScale[a.currentEpoch]

	if a.limitsComputed {
		desiredScale = a.shiftedScale[a.currentEpoch]
		logrus.Infof("Autoscaler %s using shifted scale %d with current epoch %d",
			a.revision, desiredScale, a.currentEpoch)
	} else {
		logrus.Infof("Autoscaler %s using predicted scale without limits %d with current epoch %d",
			a.revision, desiredScale, a.currentEpoch)
	}

	if desiredScale < 1 && observedConcurrency > 0 {
		desiredScale = 1
		logrus.Infof("Autoscaler %s scaling to 1 as concurrency is %f",
			a.revision, observedConcurrency)
	}

	if observedConcurrency/readyPodsCount > 2 && float64(desiredScale) <= readyPodsCount {
		prevDesired := desiredScale
		desiredScale = int32(readyPodsCount) + 1
		logrus.Infof("Autoscaler %s scaling to %d as concurrency is %f, predicted scale was %d, readyPodsCount is %f",
			a.revision, desiredScale, observedConcurrency, prevDesired, readyPodsCount)
	}

	a.currentEpoch++
	return float64(desiredScale)
}

func (a *autoscaler) ComputeInvocationsPerMinute(invocationsPerMinuteFromDataplane []float64) {
	if len(invocationsPerMinuteFromDataplane) != 60 {
		logrus.Fatal("Len of invocationsPerMinuteFromDataplane is not 60")
	}

	a.invocationsPerMinute = invocationsPerMinuteFromDataplane
}

func (a *autoscaler) EstimateCapacity(averageDurationFromDataplane int32) {
	if averageDurationFromDataplane == 0 {
		// TODO: What to do here
		// logrus.Fatal("AverageDurationFromDataplane is zero")
		averageDurationFromDataplane = 1
	}
	a.averageCapacity = 1 / float64(averageDurationFromDataplane)
}

func (a *autoscaler) predictionToScale(pred []float64) []int32 {
	predictedScale := make([]int32, 0)
	logrus.Infof("Autoscaler %s average capacity: %f at minute %d",
		a.revision, a.averageCapacity, a.currentMinute)
	for _, p := range pred {
		if p < 0.1 {
			p = 0
		}
		poissonMultiplier := 1.5
		// should account for invocation iats not being equidistant
		capacity := a.averageCapacity
		if capacity <= 0 {
			capacity = 1
		}
		s := poissonMultiplier * (1 / capacity) * p / 60
		if s > p || s <= 0 {
			s = p // scale to at most the number of invocations within that minute
		}
		if p > 0.1 && p <= 1 {
			s = 1
			// if there's only 1 invocation there's no need for more than 1 pod
		}
		scaleForMinute := make([]int32, 30) // 30 epochs within 1 minute
		for i := range scaleForMinute {
			scaleForMinute[i] = int32(math.Ceil(s))
		}
		predictedScale = append(predictedScale, scaleForMinute...)
	}
	binnedPredictedScale := binScale(3, predictedScale)
	// TODO: may need to adjust bin size
	return binnedPredictedScale
}

func binScale(binSizeInEpochs int, scalePerEpoch []int32) []int32 {
	predLength := len(scalePerEpoch)
	scalePerEpochBinned := make([]int32, predLength+1)
	for i := 0; i <= predLength-binSizeInEpochs+1; i++ {
		maxVal := scalePerEpoch[i]
		for j := 1; j < binSizeInEpochs; j++ {
			if i+j < predLength {
				maxVal = maxint32(maxVal, scalePerEpoch[i+j])
			}
		}
		for j := 0; j < binSizeInEpochs; j++ {
			if i+j < predLength+1 { // note that binned scale per epoch is longer by 1
				scalePerEpochBinned[i+j] = maxVal
			}
		}
	}
	return scalePerEpochBinned
}

func maxint32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func (a *autoscaler) oracleScaling(readyPodsCount float64, metricKey types.NamespacedName,
	now time.Time) float64 {
	var val float64
	if len(a.scale) == 0 {
		fName := a.revision
		jsonFile, err := os.Open("./var/scale_per_function/" + fName + "/scale.json")
		if err != nil {
			logrus.Infof("Couldn't open file: %s", err)
		} else {
			jsonStr, _ := ioutil.ReadAll(jsonFile)
			json.Unmarshal([]byte(jsonStr), &a.scale)
		}
		file, err := os.ReadFile("./var/time.txt")
		if err != nil {
			logrus.Infof("revision %s error when reading time.txt: %s", a.revision, err)
		}
		t := string(file)
		tInt, err := strconv.ParseInt((strings.Split(t, "\n")[0]), 10, 64)
		if err != nil {
			logrus.Infof("error when converting time.txt to integer: %s", err)
		}
		logrus.Infof("oracle revision: %s parsed time as %d", a.revision, tInt)
		a.timeToWait = tInt
	}
	if now.Unix() < a.timeToWait {
		logrus.Infof("oracle revision: %s current time: %d waiting until: %d", a.revision, now.Unix(), a.timeToWait)
		val = 0.0
	} else if a.epochCounter == len(a.scale) {
		logrus.Infof("oracle revision: %s current time: %d length equal to epoch counter", a.revision, now.Unix())
		val = 0.0
	} else {
		logrus.Infof("oracle revision: %s current time: %d waiting until: %d is over, epoch counter: %d",
			a.revision, now.Unix(), a.timeToWait, a.epochCounter)
		val = float64(a.scale[a.epochCounter])
		a.epochCounter++
	}

	logrus.Infof("oracle revision: %s, oracle time: %d, oracle desired scale: %f, oracle epoch counter: %d, oracle array length: %d",
		a.revision, now.Unix(), val, a.epochCounter, len(a.scale))
	return val
}

func (a *autoscaler) DecrementEpoch() {
	a.currentEpoch--
}

func (a *autoscaler) muAutoscaling(readyPodsCount float64, metricKey types.NamespacedName,
	now time.Time) int32 {
	if !a.startPredictiveScaling {
		logrus.Infof("Autoscaler %s checking if it can start", a.revision)
		select {
		case _ = <-a.startCh:
			a.startPredictiveScaling = true
			logrus.Infof("Autoscaler %s starting", a.revision)
		default:
			a.startPredictiveScaling = false
			// TODO: Port stuff from Drigent
			observedConcurrency, observedPanic := a.metricClient.StableAndPanicConcurrency()
			logrus.Infof("Autoscaler %s not started returning %f, panic %f", a.revision, observedConcurrency, observedPanic)
			return int32(math.Ceil(observedConcurrency))
		}
	}

	a.currentMinute = int(now.Sub(a.startTime).Minutes())

	// TODO: Port this function
	observedConcurrency, _ := a.metricClient.StableAndPanicConcurrency()

	// Dirigent takes care of it
	/*if prevMinute < a.currentMinute {
		a.computeInvocationsPerMinute(metricKey, now, logger)
		a.estimateCapacity(a.invocationsPerMinute[prevMinute], total, logger)
	}*/

	var servingRate float64 = 1
	var executionTimeInSec float64 = 1

	if a.averageCapacity > 0 {
		servingRate = a.averageCapacity
		executionTimeInSec = 1 / a.averageCapacity
	}

	// TODO: Replace this interface with Dirigent code
	observedRps := a.metricClient.StableAndPanicRPS()
	incomingRequestPerSec := observedRps

	queueLength := observedConcurrency - readyPodsCount

	var desiredPodCountSLO float64

	//SLO := spec.UserConfig.SLO
	// TODO: set SLO based on execution time
	var SLO float64 = 5
	if executionTimeInSec > 0 {
		SLO = executionTimeInSec * 10
	}

	if math.IsNaN(executionTimeInSec) {
		executionTimeInSec = 0
		fmt.Printf("error\n")
		return int32(a.prevDesiredPodCount)
	}

	if math.IsNaN(servingRate) {
		servingRate = 0
		fmt.Printf("error\n")
		return int32(a.prevDesiredPodCount)
	}

	if incomingRequestPerSec > a.MaxIRSeen {
		a.MaxIRSeen = incomingRequestPerSec
	}
	a.pattern.SingleExpectation = incomingRequestPerSec
	UpdateWeightsAll(a.modelList, &a.pattern)
	UpdatePatternInput(&a.pattern, incomingRequestPerSec)
	bestIndex := GetBestIndex(a.modelList)
	prediction := Predict(&a.modelList[bestIndex], &a.pattern)
	prediction = math.Max(prediction, 0)
	prediction = math.Min(prediction, a.MaxIRSeen)

	desiredPodCountSLO = math.Ceil(incomingRequestPerSec/servingRate) + math.Ceil(queueLength/(servingRate*(SLO-executionTimeInSec)))
	dpc := math.Max(1, desiredPodCountSLO)
	if math.IsNaN(dpc) {
		dpc = a.prevDesiredPodCount
	}
	a.prevDesiredPodCount = dpc
	logrus.Errorf("1 %s %d %d", a.revision, dpc, a.currentEpoch)
	return int32(dpc)
}
