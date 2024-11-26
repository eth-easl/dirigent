/*
 * MIT License
 *
 * Copyright (c) 2024 EASL
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

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
