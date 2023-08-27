package main

import (
	"cluster_manager/tools/shared"
	"flag"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"gonum.org/v1/gonum/stat/distuv"
)

var (
	nbColdStartsPerSecond = flag.Int("frequency", 1, "average number of colds starts per seconds")
	duration              = flag.Int("duration", 30, "duration in seconds")
	lambda                = flag.Float64("lambda", 5.0, "Lambda parameter for the poisson distribution")
)

const (
	UNIFORM_DISTRIBUTION = iota
	POISSON_DISTRIBUTION

	SECOND = 1e9
)

func main() {
	flag.Parse()

	logrus.Info("Registering services")
	shared.DeployService(&testing.T{}, (*duration)*(*nbColdStartsPerSecond), 0)

	logrus.Info("Starting sweep test")

	// Manual choose the distribution
	currentDistribution := UNIFORM_DISTRIBUTION

	switch currentDistribution {
	case UNIFORM_DISTRIBUTION:
		simulateUniformDistribution()
		break
	case POISSON_DISTRIBUTION:
		simulatePoisonDistribution()
		break
	default:
		logrus.Fatal("No distribution found")
	}

	logrus.Info("End sweep test")
}

func simulateUniformDistribution() {
	// Retrieve frequency
	trueFrequency := 1 / float64(*nbColdStartsPerSecond)

	// Start simulation
	offset := 0
	start := time.Now()

	wg := sync.WaitGroup{}
	for time.Since(start) < time.Duration(SECOND*(*duration)) {
		wg.Add(1)

		go func(offset int) {
			shared.PerformXInvocations(&testing.T{}, 1, offset)
			wg.Done()
		}(offset)

		time.Sleep(time.Duration(SECOND * trueFrequency))
		offset++
	}

	wg.Wait()
}

func simulatePoisonDistribution() {
	// We transform as follows 1) We compute total number of invocation (rate * nb of seconds) 2) We distribute them with random order
	totalInvocations := (*nbColdStartsPerSecond) * (*duration)

	// Create sampler
	poissonDistribution := distuv.Poisson{
		Lambda: *lambda,
	}

	// Sample
	wait := make([]time.Duration, totalInvocations)
	for i := 0; i < totalInvocations; i++ {
		wait[i] = time.Duration(poissonDistribution.Rand() * SECOND)
	}

	// Simulate
	wg := sync.WaitGroup{}
	wg.Add(len(wait))

	for index, timeToWait := range wait {
		go func(timeToWait time.Duration) {
			time.Sleep(timeToWait)
			shared.PerformXInvocations(&testing.T{}, 1, index)
			wg.Done()
		}(timeToWait)
	}

	// Wait until we are done
	wg.Wait()
}
