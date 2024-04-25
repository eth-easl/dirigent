package predictive_autoscaler

import (
	"github.com/mjibson/go-dsp/fft"
	"github.com/openacid/slimarray/polyfit"
	"math"
	"math/cmplx"
	"sort"
)

func fourierExtrapolation(xFloats []float64, harmonics int, nPredict int) []float64 {
	n := len(xFloats)
	harmonics = int(math.Min(float64(n-1), float64(harmonics)))
	var ta []float64
	for i := 0; i < n; i++ {
		ta = append(ta, float64(i))
	}
	f := polyfit.NewFit(ta, xFloats, 1)
	p := f.Solve()
	var xNotrend []float64
	for i := 0; i < n; i++ {
		xNotrend = append(xNotrend, xFloats[i]-p[1]*ta[i])
	}
	xFreqdom := fft.FFTReal(xNotrend)

	frequencies := make([]float64, n)
	for i := 0; i < n/2; i++ {
		frequencies[i] = float64(i) / float64(n)
	}
	for i := n / 2; i < n; i++ {
		frequencies[i] = float64((n-i)*-1) / float64(n)
	}

	var indices []int
	for i := 0; i < n; i++ {
		indices = append(indices, i)

	}
	// Sort indices by absolute value of amplitudes
	sort.Slice(indices, func(i, j int) bool {
		absI := math.Abs(real(xFreqdom[indices[i]]) + math.Abs(imag(xFreqdom[indices[i]])))
		absJ := math.Abs(real(xFreqdom[indices[j]]) + math.Abs(imag(xFreqdom[indices[j]])))
		return absI < absJ
	})

	// Reverse the slice to get descending order
	for i, j := 0, len(indices)-1; i < j; i, j = i+1, j-1 {
		indices[i], indices[j] = indices[j], indices[i]
	}

	predictions := make([]float64, nPredict)
	t := make([]float64, n+nPredict)
	for i := range t {
		t[i] = float64(i)
	}
	for i := 0; i < min(len(indices), 2*harmonics+1); i++ {
		amplitude, phase := cmplx.Polar(xFreqdom[indices[i]])
		amplitude = amplitude / float64(n)
		for j := 0; j < nPredict; j++ {
			predictions[j] += amplitude * math.Cos(2*math.Pi*frequencies[indices[i]]*t[j+n]+phase)
		}

	}
	// Add the linear trend to the predictions
	for i := range predictions {
		predictions[i] += p[1] * (t[i+n])
	}

	return predictions
}
