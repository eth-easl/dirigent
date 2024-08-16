Time required: 10 min to set up environment and 30 min per experiment
Description:  This experiment runs the downsampled Azure trace with 500 functions on Maestro. It should be repeated in two settings, with containerd and Firecracker runtimes. After you run experiment for each setting the cluster must be reloaded.

Instructions:
- Start Maestro cluster as per instructions located in the root folder of artifact evaluation instructions
- On the first node execute `mkdir ~/cluster_manager/data/traces/azure_500` and `mkdir ~/cluster_manager/data/traces/azure_500_firecracker` Copy traces `scp azure_500/* user@node0:~/invitro/data/traces/azure_500/` and `scp azure_500_firecracker/* user@node0:~/invitro/data/traces/azure_500_firecracker/`
- On node0 execute `cd ~/invitro; git checkout rps_mode`. With text editor open `cmd/config_dirigent_trace.json` and change TracePath to match
- Run locally `/scripts/start_resource_monitoring.sh user@node1 user@node2 user@node3`. 
- Run the load generator in `screen` on node0 with `cd ~/invitro; go run cmd/loader.go --config cmd/config_dirigent_trace.json`. Wait for 30 minutes. There should be around 170K invocations, with a negligible failure rate.
- Gather experient results. First, copy load generator output with `scp user@node0:/users/cvetkovi/invitro/data/out/experiment_duration_30.csv .`, and then copy resource utilization data with `mkdir cpu_mem_usage && ./scripts/collect_resource_monitoring.sh user@node1 user@node2 user@node3`. The script `collect_resource_monitoring.sh` needs to be modified on line 4 to point to directory where you want to save data (e.g., `~/azure_500/cpu_mem_usage/utilization_$1.csv`).
- Save the files you gathered somewhere, as we will use there results later.
