package managers

import (
	"fmt"
	"math"
	"testing"
)

func TestExponentialBackoff(t *testing.T) {
	eb := &ExponentialBackoff{
		Interval:        0.020,
		ExponentialRate: 1.5,
		RetryNumber:     0,
		MaxDifference:   1,
	}

	var res []float64
	for i := 0; i < 20; i++ {
		res = append(res, eb.Next())
		fmt.Println(i+1, ": ", res[i])
	}

	for i := 0; i < 11; i++ {
		if math.Abs(res[i]-res[i+1]) > eb.MaxDifference {
			t.Error("Unexpected value.")
		}
	}
}
