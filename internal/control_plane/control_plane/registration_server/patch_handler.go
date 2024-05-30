package registration_server

import (
	"cluster_manager/internal/control_plane/control_plane"
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"cluster_manager/proto"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

func patchHandler(api *control_plane.CpApiServer) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed.", http.StatusMethodNotAllowed)
			return
		}

		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Invalid parsing service registration request.", http.StatusBadRequest)
			return
		}

		function := r.FormValue("function")
		if len(function) == 0 {
			http.Error(w, "Function name not specified.", http.StatusBadRequest)
			return
		}

		api.ControlPlane.SIStorage.Lock()
		defer api.ControlPlane.SIStorage.Unlock()

		sis, ok := api.ControlPlane.SIStorage.Get(function)
		if !ok {
			http.Error(w, "Function name does not exist.", http.StatusBadRequest)
			return
		}

		oldConfig := sis.PerFunctionState.AutoscalingConfig
		config := &proto.AutoscalingConfiguration{
			ScalingUpperBound:                    oldConfig.ScalingUpperBound,
			ScalingLowerBound:                    oldConfig.ScalingLowerBound,
			PanicThresholdPercentage:             oldConfig.PanicThresholdPercentage,
			MaxScaleUpRate:                       oldConfig.MaxScaleUpRate,
			MaxScaleDownRate:                     oldConfig.MaxScaleDownRate,
			ContainerConcurrency:                 oldConfig.ContainerConcurrency,
			ContainerConcurrencyTargetPercentage: oldConfig.ContainerConcurrencyTargetPercentage,
			StableWindowWidthSeconds:             oldConfig.StableWindowWidthSeconds,
			PanicWindowWidthSeconds:              oldConfig.PanicWindowWidthSeconds,
			ScalingPeriodSeconds:                 oldConfig.ScalingPeriodSeconds,
			ScalingMethod:                        oldConfig.ScalingMethod,
		}

		if err = parseAndApplyArg("scaling_upper_bound", r, &config.ScalingUpperBound, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast scaling_upper_bound to integer - %v", err), http.StatusBadRequest)
		}
		if err = parseAndApplyArg("scaling_lower_bound", r, &config.ScalingLowerBound, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast scaling_lower_bound to integer - %v", err), http.StatusBadRequest)
		}
		if !ensureGreater(config.ScalingUpperBound, config.ScalingLowerBound) {
			http.Error(w, "Scaling upper bound cannot be lower than the scaling lower bound.", http.StatusBadRequest)
		}
		if err = parseAndApplyArg("panic_threshold_percentage", r, &config.PanicThresholdPercentage, ensurePositive[float32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast panic_threshold_percentage to float32 - %v", err), http.StatusBadRequest)
		}
		if err = parseAndApplyArg("max_scale_up_rate", r, &config.MaxScaleUpRate, ensurePositive[float32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast max_scale_up_rate to float32 - %v", err), http.StatusBadRequest)
		}
		if err = parseAndApplyArg("max_scale_down_rate", r, &config.MaxScaleDownRate, ensurePositive[float32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast max_scale_down_rate to float32 - %v", err), http.StatusBadRequest)
		}
		// TODO: propagate container concurrency change to data plane(s) or ignore changes to this parameter
		/*if err = parseAndApplyArg("container_concurrency", r, &config.ContainerConcurrency, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast container_concurrency to integer - %v", err), http.StatusBadRequest)
		}*/
		if err = parseAndApplyArg("container_concurrency_target_percentage", r, &config.ContainerConcurrencyTargetPercentage, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast container_concurrency_target_percentage to integer - %v", err), http.StatusBadRequest)
		}
		if err = parseAndApplyArg("stable_window_width", r, &config.StableWindowWidthSeconds, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast stable_window_width to integer - %v", err), http.StatusBadRequest)
		}
		if err = parseAndApplyArg("panic_window_width", r, &config.PanicWindowWidthSeconds, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast panic_window_width to integer - %v", err), http.StatusBadRequest)
		}
		if err = parseAndApplyArg("scaling_period", r, &config.ScalingPeriodSeconds, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast scaling_period to integer - %v", err), http.StatusBadRequest)
		}
		if err = parseAndApplyScalingMethod("scaling_method", r, &config.ScalingMethod); err != nil {
			http.Error(w, "Unsupported scaling_method", http.StatusBadRequest)
		}

		sis.PerFunctionState.AutoscalingConfig = config

		w.WriteHeader(http.StatusOK)
	}
}

func parseAndApplyArg[T int32 | float32](key string, r *http.Request, addr *T, validationFunc func(T) bool) error {
	if val := r.FormValue(key); len(val) != 0 {
		var iVal T
		switch any(addr).(type) {
		case *int32:
			tt, err := strconv.Atoi(val)
			if err != nil {
				return err
			}

			iVal = T(tt)
		case *float32:
			tt, err := strconv.ParseFloat(val, 32)
			if err != nil {
				return err
			}

			iVal = T(tt)
		}

		if !validationFunc(iVal) {
			return errors.New(fmt.Sprintf("key %s has invalid value", key))
		}

		*addr = iVal
	}

	// no key => no patch for field => stays the same
	return nil
}

func parseAndApplyScalingMethod(key string, r *http.Request, addr *per_function_state.AveragingMethod) error {
	if val := r.FormValue(key); len(val) != 0 {
		switch val {
		case "arithmetic":
			*addr = per_function_state.Arithmetic
		case "exponential":
			*addr = per_function_state.Exponential
		default:
			return errors.New("unknown scaling method")
		}
	}

	return nil
}

func ensurePositive[T int32 | float32](val T) bool {
	return val > 0
}

func ensureGreater(val1, val2 int32) bool {
	return val1 > val2
}
