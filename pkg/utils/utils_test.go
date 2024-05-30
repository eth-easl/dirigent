package utils

import (
	"github.com/stretchr/testify/assert"
	"math"
	"testing"
)

func TestExponentialMovingAverage(t *testing.T) {
	today := 1.0
	yesterday := 0.0

	assert.True(t, math.Abs(ExponentialMovingAverage(today, yesterday)-0.8) < 0.001)
}
