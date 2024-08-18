Time required: 10 min to set up environment and 2-3 min per data point

Description:  This experiment triggers cold start in Maestro cluster. You should sweep the load until the cluster saturates, which will be visible on the latency plot. We suggest running experiments with 1, 10, 100, 500, 1000, 1250, 1500, ... RPS and observing the latency after conducting experiment for each data point. Low RPS (<10 RPS) rates should be run for 3-5 minutes, because of warmup, while all other loads can be run for just 1 minute. Always discard the results of the first experiment when starting a new cluster, as these measurements include image pull latency, which we should not include in the results.

Instructions:
- Start Dirigent cluster according to instructions located in the root folder of artifact evaluation instructions. You can reuse the existing cluster running Dirigent containerd.
- On remote machine `node0` open `~/invitro/cmd/config_dirigent_rps.json`. Set `RpsColdStartRatioPercentage` to `100`, and sweep the load with `RpsTarget` while configuring `ExperimentDuration` according to instructions above. For higher RPS, it might be necessary to increase `RpsCooldownSeconds`, which controls the number of functions that are deployed in the cluster to achieve the requested RPS. Set `GRPCFunctionTimeoutSeconds` to `15`.
- Start RPS experiment by running `cd ~/invitro; go run cmd/loader.go --config cmd/config_dirigent_rps.json`
- Create folder storing results with `mkdir -p ./artifact_evaluation/cold_start_sweep/dirigent/results_containerd`
- Gather results located in `data/out/experiment_duration_X.csv` and copy them to your local machine in format `rps_X.csv` to the folder you created in the previous step.
- Repeat for different RPS values until the cluster saturates, which you can see by plotting the data with the provided script

Results expectation/interpretation: 
- Since we cannot provide access to a 100-node cluster over a 2-week artifact evaluation period, the throughput we show in Figure 7 is lower on smaller cluster, as worker nodes become the bottleneck. However, it is important to note that cold start throughput of Knative/K8s << Maestro - containerd < Maestro - Firecracker.
