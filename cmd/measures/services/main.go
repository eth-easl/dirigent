package main

import (
	"cluster_manager/tests/shared"
	"fmt"
	"time"
)

func main() {

	times := 0

	for times < 10 {
		counter := 0
		nbServices := 1
		// Measures
		for nbServices <= 100000 {
			start := time.Now()
			shared.DeployServiceMultiThread(nbServices, counter)
			delta := time.Since(start)
			fmt.Printf("%d,", delta)

			shared.Deregister(nbServices, counter)

			counter += nbServices
			nbServices *= 10

		}
		fmt.Println()
	}

}
