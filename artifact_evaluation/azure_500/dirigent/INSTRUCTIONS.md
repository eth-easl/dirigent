Time required: 10 min to set up environment and 30 min per experiment
Description:  This experiment runs the downsampled Azure trace with 500 functions. First run all the experiments with containerd, as given in the main `README.md`, and then deploy the cluster again, just that time with Firecracker. The procedure for running experiments is the same, just the trace with suffix `_firecracker` should be used.

Instructions:
- Start Dirigent cluster as per instructions located in the root folder of artifact evaluation instructions
- On the `node0` execute `mkdir ~/invitro/data/traces/azure_500` and `mkdir ~/invitro/data/traces/azure_500_firecracker` Copy traces `scp azure_500/* user@node0:~/invitro/data/traces/azure_500/` and `scp azure_500_firecracker/* user@node0:~/invitro/data/traces/azure_500_firecracker/`
- Make sure `~/invitro` branch is `rps_mode`. With text editor open `cmd/config_dirigent_trace.json` and change TracePath to match `azure_500` or `azure_500_firecracker`
- Run locally `./scripts/start_resource_monitoring.sh user@node1 user@node2 user@node3`. 
- Run the load generator in `screen` on `node0` with `cd ~/invitro; go run cmd/loader.go --config cmd/config_dirigent_trace.json`. Wait for 30 minutes. There should be ~170K invocations, with a negligible failure rate.
- Gather experiment results. Make sure you do not overwrite data from the other experiment.
  - Copy load generator output with `scp user@node0:~/invitro/data/out/experiment_duration_30.csv results_azure_500/`
  - Copy resource utilization data with `mkdir -p results_azure_500/cpu_mem_usage && ../scripts/collect_resource_monitoring.sh results_azure_500/cpu_mem_usage user@node1 user@node2 user@node3`.
