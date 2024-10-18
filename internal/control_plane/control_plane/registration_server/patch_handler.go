package registration_server

import (
	"cluster_manager/internal/control_plane/control_plane"
	"cluster_manager/internal/control_plane/control_plane/core"
	"cluster_manager/proto"
	"context"
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
		if sis.ServiceState.TaskInfo != nil {
			http.Error(w, "Cannot patch a task service.", http.StatusBadRequest)
		}

		config := cloneAutoscalingConfig(sis.ServiceState.GetAutoscalingConfig())

		if err = parseAndApplyArg("scaling_upper_bound", r, &config.ScalingUpperBound, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast scaling_upper_bound to integer - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("scaling_lower_bound", r, &config.ScalingLowerBound, ensurePositiveOrZero[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Cannot cast scaling_lower_bound to integer - %v", err), http.StatusBadRequest)
			return
		}
		if !ensureGreater(config.ScalingUpperBound, config.ScalingLowerBound) {
			http.Error(w, "Scaling upper bound cannot be lower than the scaling lower bound.", http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("panic_threshold_percentage", r, &config.PanicThresholdPercentage, ensurePositiveOrZero[float32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid panic_threshold_percentage - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("max_scale_up_rate", r, &config.MaxScaleUpRate, ensurePositive[float32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid max_scale_up_rate - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("max_scale_down_rate", r, &config.MaxScaleDownRate, ensurePositive[float32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid max_scale_down_rate - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("container_concurrency", r, &config.ContainerConcurrency, ensurePositiveOrZero[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid container_concurrency - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("container_concurrency_target_percentage", r, &config.ContainerConcurrencyTargetPercentage, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid container_concurrency_target_percentage - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("stable_window_width", r, &config.StableWindowWidthSeconds, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid stable_window_width - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("panic_window_width", r, &config.PanicWindowWidthSeconds, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid panic_window_width - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("scaling_period", r, &config.ScalingPeriodSeconds, ensurePositive[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid scaling_period - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyScalingMethod("scaling_method", r, &config.ScalingMethod); err != nil {
			http.Error(w, "Unsupported scaling_method", http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("target_burst_capacity", r, &config.TargetBurstCapacity, ensurePositiveOrZero[float32]); err != nil {
			http.Error(w, "Invalid target_burst_capacity", http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("initial_scale", r, &config.InitialScale, ensurePositiveOrZero[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid initial_scale - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("total_value", r, &config.TotalValue, ensurePositiveOrZero[float32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid total_value - %v", err), http.StatusBadRequest)
			return
		}
		if err = parseAndApplyArg("scale_down_delay", r, &config.ScaleDownDelay, ensurePositiveOrZero[int32]); err != nil {
			http.Error(w, fmt.Sprintf("Invalid scale_down_delay - %v", err), http.StatusBadRequest)
			return
		}
		sis.ServiceState.FunctionInfo[0].AutoscalingConfig = config

		sis.UpdateDeployment(sis.ServiceState.FunctionInfo[0])

		if err = api.ControlPlane.PersistenceLayer.StoreServiceInformation(context.Background(), sis.ServiceState.FunctionInfo[0]); err != nil {
			http.Error(w, fmt.Sprintf("Failed to persist patch handler - %v", err), http.StatusInternalServerError)
			return

		}

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

func parseAndApplyScalingMethod(key string, r *http.Request, addr *core.AveragingMethod) error {
	if val := r.FormValue(key); len(val) != 0 {
		switch val {
		case "arithmetic":
			*addr = core.Arithmetic
		case "exponential":
			*addr = core.Exponential
		default:
			return errors.New("unknown scaling method")
		}
	}

	return nil
}

func ensurePositive[T int32 | float32](val T) bool {
	return val > 0
}

func ensurePositiveOrZero[T int32 | float32](val T) bool {
	return val >= 0
}

func ensureGreater(val1, val2 int32) bool {
	return val1 > val2
}

func cloneAutoscalingConfig(cfg *proto.AutoscalingConfiguration) *proto.AutoscalingConfiguration {
	return &proto.AutoscalingConfiguration{
		ScalingUpperBound:                    cfg.ScalingUpperBound,
		ScalingLowerBound:                    cfg.ScalingLowerBound,
		PanicThresholdPercentage:             cfg.PanicThresholdPercentage,
		MaxScaleUpRate:                       cfg.MaxScaleUpRate,
		MaxScaleDownRate:                     cfg.MaxScaleDownRate,
		ContainerConcurrency:                 cfg.ContainerConcurrency,
		ContainerConcurrencyTargetPercentage: cfg.ContainerConcurrencyTargetPercentage,
		StableWindowWidthSeconds:             cfg.StableWindowWidthSeconds,
		PanicWindowWidthSeconds:              cfg.PanicWindowWidthSeconds,
		ScalingPeriodSeconds:                 cfg.ScalingPeriodSeconds,
		ScalingMethod:                        cfg.ScalingMethod,
		TargetBurstCapacity:                  cfg.TargetBurstCapacity,
		InitialScale:                         cfg.InitialScale,
		TotalValue:                           cfg.TotalValue,
		ScaleDownDelay:                       cfg.ScaleDownDelay,
	}
}
