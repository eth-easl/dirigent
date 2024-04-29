package predictive_autoscaler

import (
	"github.com/mjibson/go-dsp/fft"
	"github.com/openacid/slimarray/polyfit"
	"math"
	"math/cmplx"
	"sort"
)

func fourierExtrapolation(invocationsPerMinuteFloats []float64, harmonics int, nPredict int) []float64 {
	n := len(invocationsPerMinuteFloats)
	harmonics = int(math.Min(float64(n-1), float64(harmonics)))
	var minutesIndex []float64
	for i := 0; i < n; i++ {
		minutesIndex = append(minutesIndex, float64(i))
	}
	f := polyfit.NewFit(minutesIndex, invocationsPerMinuteFloats, 1)
	p := f.Solve()
	var invocationsPerMinuteDetrended []float64
	for i := 0; i < n; i++ {
		invocationsPerMinuteDetrended = append(invocationsPerMinuteDetrended, invocationsPerMinuteFloats[i]-p[1]*minutesIndex[i])
	}
	xFreqdom := fft.FFTReal(invocationsPerMinuteDetrended)

	frequencies := make([]float64, n)
	// TODO: this is only correct if n is even
	for i := 0; i < n/2; i++ {
		frequencies[i] = float64(i) / float64(n)
	}
	for i := n / 2; i < n; i++ {
		frequencies[i] = float64((n-i)*-1) / float64(n)
	}

	var indicesFrequency []int
	for i := 0; i < n; i++ {
		indicesFrequency = append(indicesFrequency, i)

	}
	// Sort indices by absolute value of amplitudes in descending order
	sort.Slice(indicesFrequency, func(i, j int) bool {
		absI := math.Abs(real(xFreqdom[indicesFrequency[i]]) + math.Abs(imag(xFreqdom[indicesFrequency[i]])))
		absJ := math.Abs(real(xFreqdom[indicesFrequency[j]]) + math.Abs(imag(xFreqdom[indicesFrequency[j]])))
		return absI > absJ
	})

	predictions := make([]float64, nPredict)
	predictionIndex := make([]float64, n+nPredict)
	for i := range predictionIndex {
		predictionIndex[i] = float64(i)
	}
	for i := 0; i < min(len(indicesFrequency), 2*harmonics+1); i++ {
		amplitude, phase := cmplx.Polar(xFreqdom[indicesFrequency[i]])
		amplitude = amplitude / float64(n)
		for j := 0; j < nPredict; j++ {
			predictions[j] += amplitude * math.Cos(2*math.Pi*frequencies[indicesFrequency[i]]*predictionIndex[j+n]+phase)
		}

	}
	// Add the linear trend to the predictions
	for i := range predictions {
		predictions[i] += p[1] * (predictionIndex[i+n])
	}

	return predictions
}
