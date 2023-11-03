package managers

import (
	"fmt"
	"math"
	"testing"
)

func TestExponentialBackoff(t *testing.T) {
	eb := &ExponentialBackoff{
		Interval:        0.01,
		ExponentialRate: 1.35,
		RetryNumber:     0,
		MaxDifference:   2,
	}

	dataPoints := 20

	var res []float64
	for i := 0; i < dataPoints; i++ {
		res = append(res, eb.Next())
		fmt.Println(i+1, ": ", res[i])
	}

	for i := 0; i < dataPoints-1; i++ {
		if math.Abs(res[i]-res[i+1]) > eb.MaxDifference {
			t.Error("Unexpected value.")
		}
	}
}
