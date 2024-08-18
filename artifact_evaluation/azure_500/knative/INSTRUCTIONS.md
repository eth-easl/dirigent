Time required: 10 min to set up environment and 30-60 min for the experiment

Description:  This experiment runs the downsampled Azure trace with 500 functions. Do not reuse Knative/K8s cluster if you configured the cluster for cold start sweep experiment.

Instructions:
- SSH into `node0`and on that node clone the load generator repo. Then checkout to `rps_mode` branch. The command is `git clone --branch=rps_mode https://github.com/vhive-serverless/invitro`.
- On `node0` create a directory where trace will be stored `cd invitro; mkdir data/traces/azure_500`
- Copy the trace from this folder to `node0` using the following command `scp azure_500/*.csv user@node0:~/invitro/data/traces/azure_500`
- Run locally `./scripts/start_resource_monitoring.sh user@node1 user@node2 user@node3`.
- On `node0` run `screen` and inside the screen run `go run cmd/loader.go --config cmd/config_knative.json`. Function deployment will take 10-20 minutes, and then experiment for additional 30 minutes.
- Gather experiment results. Make sure you do not overwrite data from the other experiment.
  - Copy load generator output with `scp user@node0:~/invitro/data/out/experiment_duration_30.csv results_azure_500/`
  - Copy resource utilization data with `mkdir -p ./artifact_evaluation/azure_500/knative/results_azure_500/cpu_mem_usage && ./scripts/collect_resource_monitoring.sh ./artifact_evaluation/azure_500/knative/results_azure_500/cpu_mem_usage user@node0 user@node1 user@node2`.