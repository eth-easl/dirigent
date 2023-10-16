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
