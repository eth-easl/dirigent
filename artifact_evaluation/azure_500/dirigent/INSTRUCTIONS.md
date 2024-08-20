## Azure 500 on Dirigent

Time required: 10 min to set up environment and 30 min per experiment

Description: This experiment runs the downsampled Azure trace with 500 functions. We recommend you follow the order of experiments as given in the `README.md`. 

Instructions:
- Start Dirigent cluster as per instructions located in the root folder of artifact evaluation instructions.
- On the `node0` execute `mkdir -p ~/invitro/data/traces/azure_500`.
- Copy traces from this folder to `node0` using `scp azure_500/* user@node0:~/invitro/data/traces/azure_500/`.
- Make sure on `node0` `~/invitro` branch is `rps_mode`. With text editor open `~/invitro/cmd/config_dirigent_trace.json` and change TracePath to match `data/traces/azure_500`.
- On your local machine run `./scripts/start_resource_monitoring.sh user@node0 user@node1 user@node2`. 
- Run the load generator in screen/tmux on `node0` with `cd ~/invitro; go run cmd/loader.go --config cmd/config_dirigent_trace.json`. Wait until the experiment completed (~30 minutes). There should be ~170K invocations, with a negligible failure rate.
- Gather experiment results. Make sure you do not overwrite data from the other experiment, and you place results in correct folders.
  - Create folders for storing results with `mkdir -p ./artifact_evaluation/azure_500/dirigent/results_azure_500`.
  - Copy load generator output with `scp user@node0:~/invitro/data/out/experiment_duration_30.csv results_azure_500/`
  - Copy resource utilization data with `mkdir -p ./artifact_evaluation/azure_500/dirigent/results_azure_500/cpu_mem_usage && ./scripts/collect_resource_monitoring.sh ./artifact_evaluation/azure_500/dirigent/results_azure_500/cpu_mem_usage user@node0 user@node1 user@node2`.
