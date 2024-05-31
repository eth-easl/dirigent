package predictive_autoscaler

// Pattern struct represents one pattern with dimensions and desired value
type Pattern struct {
	// Features that describe the pattern
	Features []float64
	// Numeric representation of expected value
	SingleExpectation float64
}

// ModelUnit struct represents a regression model with a slice of n weights.
type ModelUnit struct {
	// Weights represents ModelUnit vector representation
	Weights []float64
	// Bias represents ModelUnit natural propensity to spread signal
	Bias float64
	// Lrate represents learning rate of model
	Lrate float64
	//RunningError
	runningError float64
	//WindowSize
	WindowSize int
}

// UpdatePatternInput updates the input window using the new incoming rate value
func UpdatePatternInput(pattern *Pattern, newIncomingRate float64) {
	for index, value := range pattern.Features {
		pattern.Features[index] = newIncomingRate
		newIncomingRate = value
	}
}

// UpdateRunningError updates the value of runningError variable.
// It is the EWMA of the absolute error
func UpdateRunningError(model *ModelUnit, currentError float64) {
	if currentError < 0 {
		currentError = currentError * (-1.0)
	}
	model.runningError = (0.99 * model.runningError) + (0.01 * currentError)
}

// UpdateWeights performs update in model weights with respect to passed pattern.
// It returns error of prediction before and after updating weights.
func UpdateWeights(model *ModelUnit, pattern *Pattern) (float64, float64) {

	// compute prediction value and error for pattern given model BEFORE update (actual state)
	var predictedValue, prevError, postError float64 = Predict(model, pattern), 0.0, 0.0
	prevError = pattern.SingleExpectation - predictedValue

	// performs weights update for model
	model.Bias = model.Bias + model.Lrate*prevError

	// performs weights update for model
	for index, _ := range model.Weights {
		model.Weights[index] = model.Weights[index] + model.Lrate*prevError*pattern.Features[index]
	}

	// compute prediction value and error for pattern given model AFTER update (actual state)
	predictedValue = Predict(model, pattern)
	postError = pattern.SingleExpectation - predictedValue
	UpdateRunningError(model, prevError)

	// return errors
	return prevError, postError

}

// Predict performs a model prediction to passed pattern.
// It returns a float64 binary predicted value.
func Predict(model *ModelUnit, pattern *Pattern) float64 {
	var result float64 = 0.0
	// for each element compute product
	for index, value := range model.Weights {
		result = result + (value * pattern.Features[index])
	}
	return result + model.Bias
}

// Update weights for all modelUnits
func UpdateWeightsAll(modelList []ModelUnit, pattern *Pattern) {
	for index := range modelList {
		UpdateWeights(&modelList[index], pattern)
	}
}

func GetBestIndex(modelList []ModelUnit) int {
	bestIndex := 0
	minError := 999999.0
	for index := range modelList {
		if modelList[index].runningError < minError {
			minError = modelList[index].runningError
			bestIndex = index
		}
	}
	return bestIndex
}
