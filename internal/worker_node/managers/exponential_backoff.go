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

import "math"

type ExponentialBackoff struct {
	// interval * exponential_rate ^ retry_number
	Interval        float64
	ExponentialRate float64
	RetryNumber     float64

	MaxDifference float64
}

func (eb *ExponentialBackoff) calculate(retry float64) float64 {
	return eb.Interval * math.Pow(eb.ExponentialRate, retry)
}

func (eb *ExponentialBackoff) Next() float64 {
	if eb.RetryNumber == 0 {
		eb.RetryNumber += 1

		return eb.calculate(0)
	} else {
		oldValue := eb.calculate(eb.RetryNumber)
		newValue := eb.calculate(eb.RetryNumber + 1)
		eb.RetryNumber += 1

		if math.Abs(newValue-oldValue) > eb.MaxDifference {
			return -1
		} else {
			return newValue
		}
	}
}
