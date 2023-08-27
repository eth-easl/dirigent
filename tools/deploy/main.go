package main

import (
	"cluster_manager/tools/shared"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/sirupsen/logrus"
	"os"
	"testing"
)

var (
	nbServices = flag.Int("invocations", 1, "number of invocations")
	offset     = flag.Int("offset", 0, "offset")
)

func main() {
	flag.Parse()

	values := []int{1, 10, 100, 1000, 10000, 100000}
	output := make([]float64, 0)
	offset := 0

	for _, value := range values {
		var measure float64 = 0.0

		nbTests := 1

		logrus.Infof("Test value : %d", value)

		for i := 0; i < nbTests; i++ {
			logrus.Infof("Iteration : %d", i)

			duration := shared.DeployServiceTime(&testing.T{}, value, offset)
			offset += value
			measure += float64(duration) / 1e3
		}

		output = append(output, measure/float64(nbTests))
	}

	logrus.Info("Done with test")

	{
		file, err := os.Create("output.txt")
		if err != nil {
			panic(err)
		}

		b, _ := json.Marshal(output)
		fmt.Println(string(b))
		file.WriteString(string(b))
	}

	fmt.Println(output)
}
