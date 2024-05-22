package autoscaling

import (
	"cluster_manager/proto"
	"github.com/montanaflynn/stats"
	"github.com/sirupsen/logrus"
	"strconv"
	"sync"
	"testing"
	"time"
)

func TestAutoscalingPerformance(t *testing.T) {
	tests := []struct {
		Name        string
		Iterations  int
		Parallelism int
	}{
		{
			Name:        "10_function",
			Iterations:  1,
			Parallelism: 10,
		},
		{
			Name:        "100_function",
			Iterations:  1,
			Parallelism: 100,
		},
		{
			Name:        "1000_function",
			Iterations:  1,
			Parallelism: 1000,
		},
		{
			Name:        "10000_function",
			Iterations:  1,
			Parallelism: 10000,
		},
		{
			Name:        "100000_function",
			Iterations:  1,
			Parallelism: 100000,
		},
		{
			Name:        "1000000_function",
			Iterations:  1,
			Parallelism: 1000000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.Name, func(t *testing.T) {
			logrus.SetLevel(logrus.InfoLevel)
			logrus.SetFormatter(&logrus.TextFormatter{TimestampFormat: time.StampMilli, FullTimestamp: true})

			var data []float64
			wg := sync.WaitGroup{}
			mutex := sync.Mutex{}

			for iter := 0; iter < tt.Iterations; iter++ {
				channels := make([]chan int, tt.Parallelism)
				services := make([]*proto.ServiceInfo, tt.Parallelism)
				controllers := make([]*PFStateController, tt.Parallelism)

				wg.Add(tt.Parallelism)
				for i := 0; i < tt.Parallelism; i++ {
					channels[i] = make(chan int)
					services[i] = &proto.ServiceInfo{
						Name:              "service_" + strconv.Itoa(i),
						AutoscalingConfig: NewDefaultAutoscalingMetadata(),
					}
					controllers[i] = NewPerFunctionStateController(channels[i], services[i], 2*time.Second)

					go func(idx int) {
						defer wg.Done()

						start := time.Now()
						controllers[idx].Start()
						<-channels[idx]
						dt := time.Since(start).Microseconds()

						mutex.Lock()
						data = append(data, float64(dt))
						mutex.Unlock()

						logrus.Tracef("Autoscaling took %d ms", dt)
					}(i)
				}

				wg.Wait()
			}

			p50, _ := stats.Percentile(data, 50)
			p99, _ := stats.Percentile(data, 99)
			logrus.Infof("Parallelism: %d; p50: %.2f μs; p99: %.2f μs", tt.Parallelism, p50, p99)
		})
	}
}
