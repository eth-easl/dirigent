package control_plane

import (
	"cluster_manager/internal/control_plane/control_plane/per_function_state"
	"cluster_manager/internal/control_plane/control_plane/predictive_autoscaler"
	"cluster_manager/pkg/config"
	"cluster_manager/pkg/utils"
	"cluster_manager/proto"
	"context"
	"github.com/montanaflynn/stats"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestAutoscalingPerformance(t *testing.T) {
	logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

	tests := []struct {
		Name        string
		Iterations  int
		Parallelism int
		Policy      string
	}{
		{
			Name:        "mu_10",
			Iterations:  10,
			Parallelism: 10,
			Policy:      utils.MU_AUTOSCALER,
		},
		{
			Name:        "mu_100",
			Iterations:  10,
			Parallelism: 100,
			Policy:      utils.MU_AUTOSCALER,
		},
		{
			Name:        "predictive_10",
			Iterations:  10,
			Parallelism: 10,
			Policy:      utils.MU_AUTOSCALER,
		},
		{
			Name:        "predictive_100",
			Iterations:  5,
			Parallelism: 100,
			Policy:      utils.MU_AUTOSCALER,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			ctx := context.Background()

			cfg := &config.ControlPlaneConfig{
				Autoscaler: tt.Policy,
			}

			autoscalingConfig := &proto.AutoscalingConfiguration{
				MaxScaleUpRate:           1000,
				MaxScaleDownRate:         2,
				TotalValue:               10000,
				PanicThresholdPercentage: 200,
				StableWindowWidthSeconds: 60,
				ScaleDownDelay:           2,
			}

			var data []float64
			wg := sync.WaitGroup{}
			mutex := sync.Mutex{}

			for iter := 0; iter < tt.Iterations; iter++ {
				multiscaler := predictive_autoscaler.NewMultiScaler(cfg, uniscalerFactoryCreator)

				services := make([]*proto.ServiceInfo, tt.Parallelism)
				states := make([]*per_function_state.PFState, tt.Parallelism)

				wg.Add(tt.Parallelism)
				for i := 0; i < tt.Parallelism; i++ {
					services[i] = &proto.ServiceInfo{
						Name:              "service_" + strconv.Itoa(i),
						AutoscalingConfig: per_function_state.NewDefaultAutoscalingMetadata(),
					}
					states[i] = per_function_state.NewPerFunctionState(services[i])

					_, _ = multiscaler.Create(ctx, states[i],
						&predictive_autoscaler.Decider{
							Name:                     services[i].Name,
							AutoscalingConfiguration: autoscalingConfig,
							Status:                   predictive_autoscaler.DeciderStatus{},
						},
					)

					go func(idx int) {
						defer wg.Done()

						start := time.Now()
						multiscaler.Poke(services[idx].Name, 0)

						scaler := multiscaler.GetRawScaler(services[idx].Name)
						<-scaler.GetUniScaler().GetDesiredStateChannel()
						dt := time.Since(start).Microseconds()

						mutex.Lock()
						data = append(data, float64(dt))
						mutex.Unlock()

						logrus.Debugf("Autoscaling took %d μs", dt)
					}(i)
				}
			}

			wg.Wait()

			p50, _ := stats.Percentile(data, 50)
			p99, _ := stats.Percentile(data, 99)
			logrus.Infof("Parallelism: %d; p50: %.2f μs; p99: %.2f μs", tt.Parallelism, p50, p99)
		})
	}
}
