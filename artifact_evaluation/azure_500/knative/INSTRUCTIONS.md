## Azure 500 on Knative/K8s

Time required: 10 min to set up environment and 30-60 min for the experiment

Description: This experiment runs the downsampled Azure trace with 500 functions. Do not reuse Knative/K8s cluster if you configured the cluster for cold start sweep experiment. 

Important: Do not reuse Knative/K8s cluster if you previously ran cold start sweep experiments, as the autoscaling configuration was changed and could affect the results severely.

Instructions:
- SSH into `node0` and on that node clone the load generator repo. Then checkout to `rps_mode` branch. The command is `git clone --branch=rps_mode https://github.com/vhive-serverless/invitro`.
- On `node0` create a directory where trace will be stored `cd invitro; mkdir data/traces/azure_500`.
- Copy the trace from folder where this instruction file is located to the folder you previously created on `node0` using the following command `scp azure_500/*.csv user@node0:~/invitro/data/traces/azure_500`. 
- On your local machine run `./scripts/start_resource_monitoring.sh user@node0 user@node1 user@node2`.
- On `node0` inside screen/tmux run `cd ~/invitro; go run cmd/loader.go --config cmd/config_knative.json`. Function deployment will take 10-20 minutes, and then experiment will run for additional 30 minutes.
- Gather experiment results. Make sure you do not overwrite data from the other experiment, and you place results in correct folders.
  - Create a folder for storing results with `mkdir -p ./artifact_evaluation/azure_500/knative/results_azure_500`
  - Copy load generator output with `scp user@node0:~/invitro/data/out/experiment_duration_30.csv results_azure_500/`
  - Copy resource utilization data with `mkdir -p ./artifact_evaluation/azure_500/knative/results_azure_500/cpu_mem_usage && ./scripts/collect_resource_monitoring.sh ./artifact_evaluation/azure_500/knative/results_azure_500/cpu_mem_usage user@node0 user@node1 user@node2`.