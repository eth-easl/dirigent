/*
Copyright 2018 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package predictive_autoscaler

import (
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"cluster_manager/proto"
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"math"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// tickInterval is how often the Autoscaler evaluates the metrics
// and issues a decision.
const tickInterval = 2 * time.Second

type ScalingDecisions struct {
	scale []int32
}

// Decider is a resource which observes the request load of a Revision and
// recommends a number of replicas to run.
// +k8s:deepcopy-gen=true
type Decider struct {
	metav1.ObjectMeta
	Spec   DeciderSpec
	Status DeciderStatus
}

// DeciderSpec is the parameters by which the Revision should be scaled.
type DeciderSpec struct {
	MaxScaleUpRate   float64
	MaxScaleDownRate float64
	// The metric used for scaling, i.e. concurrency, rps.
	ScalingMetric string
	// The value of scaling metric per pod that we target to maintain.
	// TargetValue <= TotalValue.
	TargetValue float64
	// The total value of scaling metric that a pod can maintain.
	TotalValue float64
	// The burst capacity that user wants to maintain without queuing at the POD level.
	// Note, that queueing still might happen due to the non-ideal load balancing.
	TargetBurstCapacity float64
	// ActivatorCapacity is the single activator capacity, for subsetting.
	ActivatorCapacity float64
	// PanicThreshold is the threshold at which panic mode is entered. It represents
	// a factor of the currently observed load over the panic window over the ready
	// pods. I.e. if this is 2, panic mode will be entered if the observed metric
	// is twice as high as the current population can handle.
	PanicThreshold float64
	// StableWindow is needed to determine when to exit panic mode.
	StableWindow time.Duration
	// ScaleDownDelay is the time that must pass at reduced concurrency before a
	// scale-down decision is applied.
	ScaleDownDelay time.Duration
	// InitialScale is the calculated initial scale of the revision, taking both
	// revision initial scale and cluster initial scale into account. Revision initial
	// scale overrides cluster initial scale.
	InitialScale int32
	// Reachable describes whether the revision is referenced by any route.
	Reachable bool
	// ActivationScale is the minimum, non-zero value that a service should scale to.
	// For example, if ActivationScale = 2, when a service scaled from zero it would
	// scale up two replicas in this case. In essence, this allows one to set both a
	// min-scale value while also preserving the ability to scale to zero.
	// ActivationScale must be >= 2.
	ActivationScale int32
}

// DeciderStatus is the current scale recommendation.
type DeciderStatus struct {
	// DesiredScale is the target number of instances that autoscaler
	// this revision needs.
	DesiredScale int32

	// ExcessBurstCapacity is the difference between spare capacity
	// (how much more load the pods in the revision deployment can take before being
	// overloaded) and the configured target burst capacity.
	// If this number is negative: Activator will be threaded in
	// the request path by the PodAutoscaler controller.
	ExcessBurstCapacity int32
}

// ScaleResult holds the scale result of the UniScaler evaluation cycle.
type ScaleResult struct {
	// DesiredPodCount is the number of pods Autoscaler suggests for the revision.
	DesiredPodCount int32
	// ExcessBurstCapacity is computed headroom of the revision taking into
	// the account target burst capacity.
	ExcessBurstCapacity int32
	// ScaleValid specifies whether this scale result is valid, i.e. whether
	// Autoscaler had all the necessary information to compute a suggestion.
	ScaleValid bool
}

var invalidSR = ScaleResult{
	ScaleValid: false,
}

// UniScaler records statistics for a particular Decider and proposes the scale for the Decider's target based on those statistics.
type UniScaler interface {
	// Scale computes a scaling suggestion for a revision.
	Scale(time.Time) ScaleResult

	// Update reconfigures the UniScaler according to the DeciderSpec.
	Update(*DeciderSpec)

	SharePredictions()

	LimitScalingDecisions() ([]int32, bool)

	GetDesiredStateChannel() chan int

	ComputeInvocationsPerMinute(invocationsPerMinuteFromDataplane []float64)

	EstimateCapacity(averageDurationFromDataplane int32)
}

// UniScalerFactory creates a UniScaler for a given PA using the given dynamic configuration.
// Unique to Dirigent, we add the desired state channel
type UniScalerFactory func(*per_function_state.PFState, *Decider, chan ScalingDecisions, chan ScalingDecisions, chan bool) (UniScaler, error)

// scalerRunner wraps a UniScaler and a channel for implementing shutdown behavior.
type scalerRunner struct {
	scaler           UniScaler
	stopCh           chan struct{}
	pokeCh           chan struct{}
	predictionsCh    chan ScalingDecisions
	shiftedScalingCh chan ScalingDecisions
	startCh          chan bool

	scalePerMinute        []int32
	limitedScalePerMinute []int32
	predictionUpdateTime  time.Time

	// mux guards access to decider.
	mux     sync.RWMutex
	decider *Decider
}

func sameSign(a, b int32) bool {
	return (a&math.MinInt32)^(b&math.MinInt32) == 0
}

// decider returns a thread safe deep copy of the owned decider.
func (sr *scalerRunner) safeDecider() *Decider {
	sr.mux.RLock()
	defer sr.mux.RUnlock()
	// TODO: What to do here?
	return sr.decider.DeepCopy()
}

func (sr *scalerRunner) updateLatestScale(sRes ScaleResult) bool {
	ret := false
	sr.mux.Lock()
	defer sr.mux.Unlock()
	if sr.decider.Status.DesiredScale != sRes.DesiredPodCount {
		sr.decider.Status.DesiredScale = sRes.DesiredPodCount
		ret = true
	}

	// If sign has changed -- then we have to update KPA.
	ret = ret || !sameSign(sr.decider.Status.ExcessBurstCapacity, sRes.ExcessBurstCapacity)

	// Update with the latest calculation anyway.
	sr.decider.Status.ExcessBurstCapacity = sRes.ExcessBurstCapacity
	return ret
}

// MultiScaler maintains a collection of UniScalers.
type MultiScaler struct {
	scalersMutex sync.RWMutex
	scalers      map[string]*scalerRunner

	scalersStopCh <-chan struct{}

	uniScalerFactory UniScalerFactory

	watcherMutex sync.RWMutex
	watcher      func(string)

	tickProvider func(time.Duration) *time.Ticker

	limitsComputed time.Time

	startAutoscalers bool
}

// NewMultiScaler constructs a MultiScaler.
func NewMultiScaler(
	stopCh <-chan struct{},
	uniScalerFactory UniScalerFactory) *MultiScaler {
	return &MultiScaler{
		scalersMutex:     sync.RWMutex{},
		scalers:          make(map[string]*scalerRunner),
		scalersStopCh:    stopCh,
		uniScalerFactory: uniScalerFactory,
		tickProvider:     time.NewTicker,
	}
}

// Get returns the copy of the current Decider.
func (m *MultiScaler) Get(_ context.Context, namespace, name string) (*Decider, error) {
	m.scalersMutex.RLock()
	defer m.scalersMutex.RUnlock()
	scaler, exists := m.scalers[name]
	if !exists {
		// This GroupResource is a lie, but unfortunately this interface requires one.
		// TODO: What to do here
		return nil, errors.New("Scaler isn't present")
	}
	return scaler.safeDecider(), nil
}

// Create instantiates the desired Decider.
// Last parameter is unique to Dirigent
func (m *MultiScaler) Create(_ context.Context, functionState *per_function_state.PFState, decider *Decider) (*Decider, error) {
	m.scalersMutex.Lock()
	defer m.scalersMutex.Unlock()
	scaler, exists := m.scalers[decider.Name]
	if !exists {
		var err error
		logrus.Warnf("%s", decider.Name)
		scaler, err = m.createScaler(functionState, decider, decider.Name)
		if err != nil {
			return nil, err
		}
		m.scalers[decider.Name] = scaler
	}
	return scaler.safeDecider(), nil
}

// Update applies the desired DeciderSpec to a currently running Decider.
func (m *MultiScaler) Update(_ context.Context, decider *Decider) (*Decider, error) {
	m.scalersMutex.Lock()
	defer m.scalersMutex.Unlock()
	if scaler, exists := m.scalers[decider.Name]; exists {
		scaler.mux.Lock()
		defer scaler.mux.Unlock()
		// Make sure we store the copy.
		scaler.decider = decider.DeepCopy()
		scaler.scaler.Update(&decider.Spec)
		return decider, nil
	}
	// This GroupResource is a lie, but unfortunately this interface requires one.
	return nil, errors.New("Scaler isn't present")
}

// TODO: This is really hugly, at some point remove
func (m *MultiScaler) intToFloat(data []uint32) []float64 {
	output := make([]float64, len(data))
	for i, value := range data {
		output[i] = float64(value)
	}
	return output
}

// Unique to Dirigent
func (m *MultiScaler) ForwardDataplaneMetrics(dataplaneMetrics *proto.MetricsPredictiveAutoscaler) error {
	m.scalersMutex.Lock()
	defer m.scalersMutex.Unlock()

	for i := 0; i < len(dataplaneMetrics.GetFunctionDuration()); i++ {
		// TODO: Fix this type error
		if scaler, exists := m.scalers[dataplaneMetrics.GetFunctionNames()[i]]; exists {
			// TODO: Call somehow the two functions here
			// TODO: Change type in data plane
			// TODO: Change data types in the autoscaler
			scaler.scaler.EstimateCapacity(int32(dataplaneMetrics.GetFunctionDuration()[i]))
			// TODO: Change type in the autoscaler
			sliceToSend := dataplaneMetrics.GetInvocationsPerMinute()[60*i : 60*(i+1)]
			scaler.scaler.ComputeInvocationsPerMinute(m.intToFloat(sliceToSend))
		}
	}

	return nil
}

// Delete stops and removes a Decider.
func (m *MultiScaler) Delete(_ context.Context, namespace, name string) {
	m.scalersMutex.Lock()
	defer m.scalersMutex.Unlock()
	if scaler, exists := m.scalers[name]; exists {
		close(scaler.stopCh)
		delete(m.scalers, name)
	}
}

// Inform sends an update to the registered watcher function, if it is set.
func (m *MultiScaler) Inform(event string) bool {
	m.watcherMutex.RLock()
	defer m.watcherMutex.RUnlock()

	if m.watcher != nil {
		m.watcher(event)
		return true
	}
	return false
}

func (m *MultiScaler) runScalerTicker(runner *scalerRunner, metricKey string) {
	ticker := m.tickProvider(tickInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-m.scalersStopCh:
				return
			case <-runner.stopCh:
				return
			case <-ticker.C:
				m.tickScaler(runner.scaler, runner, metricKey)
			case <-runner.pokeCh:
				m.tickScaler(runner.scaler, runner, metricKey)
			}
		}
	}()
}

func (m *MultiScaler) createScaler(functionState *per_function_state.PFState, decider *Decider, key string) (*scalerRunner, error) {
	d := decider.DeepCopy()
	predictionsCh := make(chan ScalingDecisions, 5)
	shiftedScalingCh := make(chan ScalingDecisions, 5)
	startCh := make(chan bool, 5)
	scaler, err := m.uniScalerFactory(functionState, d, predictionsCh, shiftedScalingCh, startCh)
	if err != nil {
		return nil, err
	}

	runner := &scalerRunner{
		scaler:           scaler,
		stopCh:           make(chan struct{}),
		decider:          d,
		pokeCh:           make(chan struct{}),
		predictionsCh:    predictionsCh,
		shiftedScalingCh: shiftedScalingCh,
		startCh:          startCh,
	}
	d.Status.DesiredScale = -1
	switch tbc := d.Spec.TargetBurstCapacity; tbc {
	case -1, 0:
		d.Status.ExcessBurstCapacity = int32(tbc)
	default:
		// If TBC > Target * InitialScale, then we know initial
		// scale won't be enough to cover TBC and we'll be behind activator.
		d.Status.ExcessBurstCapacity = int32(float64(d.Spec.InitialScale)*d.Spec.TotalValue - tbc)
	}
	m.startAutoscalers = false
	m.runScalerTicker(runner, key)
	return runner, nil
}

func (m *MultiScaler) tickScaler(scaler UniScaler, runner *scalerRunner, metricKey string) {
	go m.receivePredictions(runner)
	sr := scaler.Scale(time.Now())

	// Unique to Dirigent
	tunnelToScalingLoop := scaler.GetDesiredStateChannel()
	// Forward values to scaling loop in Dirigent
	tunnelToScalingLoop <- int(sr.DesiredPodCount)

	if !sr.ScaleValid {
		logrus.Info("Multiscaler got invalid scale")
		return
	}

	if !m.startAutoscalers && sr.DesiredPodCount > 0 {
		if ok := m.scalersMutex.TryLock(); ok {
			defer m.scalersMutex.Unlock()
			for _, r := range m.scalers {
				r.startCh <- true
			}
			m.startAutoscalers = true
			logrus.Info("Multiscaler started all autoscalers")
		}
	}

	if runner.updateLatestScale(sr) {
		m.Inform(metricKey)
	}

}

func (m *MultiScaler) receivePredictions(runner *scalerRunner) {
	if m.startAutoscalers {
		var pred []int32
		select {
		case s := <-runner.predictionsCh:
			pred = s.scale
			logrus.Infof("Multiscaler got predictions of length %d", len(pred))
		default:
			pred = nil
		}
		if pred != nil {
			runner.scalePerMinute = pred
			runner.predictionUpdateTime = time.Now()
			logrus.Info("Multiscaler updated predictions")
		}

		if ok := m.scalersMutex.TryLock(); ok {
			m.checkIfGlobalScalingLimitsCanBeComputed()
		}
	}
}

func (m *MultiScaler) Poke(key string) {
	m.scalersMutex.RLock()
	defer m.scalersMutex.RUnlock()

	scaler, exists := m.scalers[key]
	if !exists {
		return
	}

	// Tick here
	scaler.pokeCh <- struct{}{}
}

func (m *MultiScaler) checkIfGlobalScalingLimitsCanBeComputed() {
	defer m.scalersMutex.Unlock()

	allPredictionsComputed := time.Now()
	mostRecentPrediction := time.Time{}

	for _, r := range m.scalers {
		if r.predictionUpdateTime.Before(allPredictionsComputed) {
			allPredictionsComputed = r.predictionUpdateTime
		}
		if mostRecentPrediction.Before(r.predictionUpdateTime) {
			mostRecentPrediction = r.predictionUpdateTime
		}
	}
	minutesDiffMostRecentLeastRecent := int(mostRecentPrediction.Sub(allPredictionsComputed).Minutes())
	minutesSinceLimitsComputed := int(time.Now().Sub(m.limitsComputed).Minutes())
	logrus.Infof("all pred computed %t, minutes since limits computed: %d, minutes diff: %d",
		allPredictionsComputed.IsZero(), minutesSinceLimitsComputed, minutesDiffMostRecentLeastRecent)
	if !allPredictionsComputed.IsZero() && minutesSinceLimitsComputed > 15 && minutesDiffMostRecentLeastRecent < 10 {
		logrus.Info("Multiscaler computing global scaling limits")
		m.computeGlobalScalingLimits()
		m.limitsComputed = time.Now()
		for _, r := range m.scalers {
			r.shiftedScalingCh <- ScalingDecisions{scale: r.limitedScalePerMinute}
			logrus.Infof("Multiscaler sending predictions of length %d", len(r.limitedScalePerMinute))
		}
	}
}

func (m *MultiScaler) getGlobalThreshold() int32 {
	return 80
}

func (m *MultiScaler) computeGlobalScalingLimits() {
	limitComputationStartTime := time.Now()
	scalePerFunctionBinned := make(map[string][]int32)
	for f, r := range m.scalers {
		scalePerFunctionBinned[f] = r.scalePerMinute
		logrus.Infof("Adding key %s with length %d to scale per function",
			f, len(r.scalePerMinute))
	}
	totalUpscaling, totalDownscaling := m.computeTotalScalingDecisions(scalePerFunctionBinned)
	logrus.Infof("totalUpscaling length %d, totalDownscaling length %d",
		len(totalUpscaling), len(totalDownscaling))
	var threshold int32 = m.getGlobalThreshold()
	m.shiftUpscaling(threshold, totalUpscaling, scalePerFunctionBinned)
	totalUpscaling, totalDownscaling = m.computeTotalScalingDecisions(scalePerFunctionBinned)
	logrus.Infof("totalUpscaling length %d, totalDownscaling length %d after shiting upscaling",
		len(totalUpscaling), len(totalDownscaling))
	m.shiftDownscaling(threshold, totalDownscaling, scalePerFunctionBinned)
	for f, s := range scalePerFunctionBinned {
		m.scalers[f].limitedScalePerMinute = s
		logrus.Infof("Setting limited scale with length %d for %s", len(s), f)
		logrus.Infof("Initial prediction was %d", m.scalers[f].scalePerMinute)
		logrus.Infof("Result is %d", s)
	}
	limitComputationEndTime := time.Now().Sub(limitComputationStartTime)
	logrus.Infof("Limit computation time %d nanoseconds", limitComputationEndTime.Nanoseconds())
}

func (m *MultiScaler) computeTotalScalingDecisions(scalePerFunction map[string][]int32) ([]int32, []int32) {
	functions := make([]string, len(scalePerFunction))
	i := 0
	for f := range scalePerFunction {
		functions[i] = f
		i++
	}
	predictionWindow := len(scalePerFunction[functions[0]])
	logrus.Infof("predictionWindow length %d", predictionWindow)
	totalUpscaling := make([]int32, predictionWindow)
	totalDownscaling := make([]int32, predictionWindow)
	for _, f := range functions {
		if len(scalePerFunction[f]) < predictionWindow {
			scalePerFunction[f] = append(scalePerFunction[f], scalePerFunction[f][len(scalePerFunction[f])-1])
			logrus.Infof("Increasing length of scale per function %s that has length %d, "+
				"as prediction window has length %d", f, len(scalePerFunction[f]), predictionWindow)
		}
		totalUpscaling[0] += scalePerFunction[f][0]
		for i := 0; i < predictionWindow-1; i++ {
			totalUpscaling[i+1] += int32(math.Max(0, float64(scalePerFunction[f][i+1]-scalePerFunction[f][i])))
			totalDownscaling[i+1] += int32(math.Max(0, float64(scalePerFunction[f][i]-scalePerFunction[f][i+1])))
		}
	}
	return totalUpscaling, totalDownscaling
}

func (m *MultiScaler) shiftUpscaling(threshold int32, totalUpscaling []int32,
	scalePerFunctionBinned map[string][]int32) {
	functions := make([]string, len(scalePerFunctionBinned))
	j := 0
	for f := range scalePerFunctionBinned {
		functions[j] = f
		j++
	}

	for i := len(totalUpscaling) - 1; i > 0; i-- {
		if totalUpscaling[i] > threshold {
			functionsToUpscale := make(map[string]int32)
			for _, f := range functions {
				if scalePerFunctionBinned[f][i] > scalePerFunctionBinned[f][i-1] {
					functionsToUpscale[f] = scalePerFunctionBinned[f][i] - scalePerFunctionBinned[f][i-1]
				}
			}
			sumFunctionsToUpscale := float64(sumValues(functionsToUpscale))
			logrus.Infof("functionsToUpscale has length %d with sum %f",
				len(functionsToUpscale), sumFunctionsToUpscale)
			totalUpscalingAllocation := int(math.Floor(float64(threshold)))
			totalUpscalingAllocation = int(math.Min(float64(totalUpscalingAllocation), sumFunctionsToUpscale))
			// set upscaling for all functions to 0
			for f := range functionsToUpscale {
				scalePerFunctionBinned[f][i-1] = scalePerFunctionBinned[f][i]
			}
			// iterate over functions and increase their upscaling one by one
			for totalUpscalingAllocation > 0 {
				for f := range functionsToUpscale {
					if totalUpscalingAllocation >= 1 && functionsToUpscale[f] > 0 {
						scalePerFunctionBinned[f][i-1]--
						totalUpscalingAllocation--
						functionsToUpscale[f]--
					} else if totalUpscalingAllocation >= 1 {
						continue
					} else {
						break
					}
				}
			}
			// recompute totalUpscaling
			totalUpscaling[i-1] = 0
			totalUpscaling[i] = 0
			for _, f := range functions {
				if i > 1 {
					totalUpscaling[i-1] += int32(math.Max(0, float64(scalePerFunctionBinned[f][i-1]-scalePerFunctionBinned[f][i-2])))
				}
				totalUpscaling[i] += int32(math.Max(0, float64(scalePerFunctionBinned[f][i]-scalePerFunctionBinned[f][i-1])))
			}
		}
	}
}

func (m *MultiScaler) shiftDownscaling(threshold int32, totalDownscaling []int32,
	scalePerFunctionBinned map[string][]int32) {
	functions := make([]string, len(scalePerFunctionBinned))
	j := 0
	for f := range scalePerFunctionBinned {
		functions[j] = f
		j++
	}

	for i := 0; i < len(totalDownscaling)-1; i++ {
		if totalDownscaling[i] > threshold {
			functionsToDownscale := make(map[string]int32)
			for _, f := range functions {
				if scalePerFunctionBinned[f][i] > scalePerFunctionBinned[f][i+1] {
					functionsToDownscale[f] = scalePerFunctionBinned[f][i] - scalePerFunctionBinned[f][i+1]
				}
			}
			sumFunctionsToDownscale := float64(sumValues(functionsToDownscale))
			logrus.Infof("functionsToDownscale has length %d with sum %f",
				len(functionsToDownscale), sumFunctionsToDownscale)
			totalDownscalingAllocation := int(math.Floor(float64(threshold)))
			totalDownscalingAllocation = int(math.Min(float64(totalDownscalingAllocation), sumFunctionsToDownscale))
			// set downscaling for all functions to 0
			for f := range functionsToDownscale {
				scalePerFunctionBinned[f][i+1] = scalePerFunctionBinned[f][i]
			}
			// iterate over functions and increase their downscaling one by one
			for totalDownscalingAllocation > 0 {
				for f := range functionsToDownscale {
					if totalDownscalingAllocation >= 1 && functionsToDownscale[f] > 0 {
						scalePerFunctionBinned[f][i+1]--
						totalDownscalingAllocation--
						functionsToDownscale[f]--
					} else if totalDownscalingAllocation >= 1 {
						continue
					} else {
						break
					}
				}
			}
			// recompute totalDownscaling
			totalDownscaling[i+1] = 0
			totalDownscaling[i] = 0
			for _, f := range functions {
				totalDownscaling[i+1] += int32(math.Max(0, float64(scalePerFunctionBinned[f][i]-scalePerFunctionBinned[f][i+1])))
				if i > 0 {
					totalDownscaling[i] += int32(math.Max(0, float64(scalePerFunctionBinned[f][i-1]-scalePerFunctionBinned[f][i])))
				}
			}
		}
	}
}

func sumValues(data map[string]int32) int32 {
	var sum int32 = 0
	for _, value := range data {
		sum += value
	}
	return sum
}
