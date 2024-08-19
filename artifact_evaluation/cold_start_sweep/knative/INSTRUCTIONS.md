## Cold start sweep on Knative/K8s

Time required: 10 min to set up environment and 2-3 min per data point

Description:  This experiment triggers cold start in Maestro cluster. You should sweep the load until the cluster saturates, which will be visible on the latency plot and should happen around 3 RPS. We suggest running experiments with 1, 2, 3 RPS and observing the latency after conducting experiment for each data point. Always discard the results of the first experiment when starting a new cluster, as these measurements include image pull latency, which pollutes the results.

Instructions: 
- Start Knative/K8s cluster according to instructions located in the root folder of artifact evaluation instructions (`README.md`). You can reuse the existing cluster running Knative/K8s, but after executing the instructions below do not use such cluster for running Azure 500 trace again.
- On `node0` execute the following commands:
  - Open `~/invitro/workloads/container/trace_func_go.yaml`, set `autoscaling.knative.dev/max-scale` to `1`, and then set image to `docker.io/cvetkovic/dirigent_empty_function:latest`.
  - Run `kubectl patch configmap config-autoscaler -n knative-serving -p '{"data":{"scale-to-zero-grace-period":"1s","scale-to-zero-pod-retention-period":"1s","stable-window":"6s"}}'`
  - In `~/invitro/cmd/config_knative_rps.json` set `ExperimentDuration` to 2 and `RpsColdStartRatioPercentage` to `100`
- The command for running experiment for each data point is `cd invitro; go run cmd/loader.go --config cmd/config_knative_rps.json`. Use the following data point settings in `cmd/config_knative_rps.json` for experiments.
  - `RpsTarget=1` with `RpsCooldownSeconds=10`
  - `RpsTarget=2` with `RpsCooldownSeconds=15`
  - `RpsTarget=3` with `RpsCooldownSeconds=20`
- After each experiment copy results from `data/out/experiment_duration_2.csv` and save it to your local machine at `artifact_evaluation/cold_start_sweep/knative/results/rps_X.csv`