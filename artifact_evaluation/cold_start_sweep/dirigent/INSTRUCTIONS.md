Time required: 10 min to set up environment and 2-3 min per data point
Description:  This experiment triggers cold start in Maestro cluster. You should sweep the RPS until the cluster saturates. We suggest running experiments with 1, 10, 100, 500, 1000, 1500, ... RPS and observing the plot after each gathering data for each data point. Low RPS rates should run for 3-5 minutes, due to warmup periods, whereas for anything above 100 RPS, you can run an experiment for 1 minute.

Instructions:
- Start Maestro cluster as per instructions located in the root folder of artifact evaluation instructions
- Edit on the remote node `~/invitro/cmd/config_dirigent_rps.json`. Set RpsColdStartRatioPercentage=100, and use RpsTarget and ExperimentDuration while sweeping the RPS. For higher RPS, it might be necessary to increase RPsCooldownSeconds, which controls the number of functions that are deployed in the cluster to achieve the requested RPS.
- Start RPS experiment by running `go run cmd/loader.go --config cmd/config_dirigent_rps.json`
- Gather results located in `data/out/experiment_duration_X.csv` and copy them to your local machine in format `rps_X.csv`
- Repeat for different RPS values until the cluster saturates, which you can see by plotting the data with the provided script

Results to expect: 
- Because we cannot provide access to 100-node cluster over a 2-week period, expect that the cluster achieves a lower RPS than what is described in the paper. However, it is important that Knative/K8s << Maestro - containerd < Maestro - Firecracker cold start throughput.
