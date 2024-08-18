## Dirigent Artifact Evaluation Instructions

The following experiments aim to repeat results from Figures 7, 9, and 10, i.e., main results from sections 5.3 and 5.2.1, which themselves constitute the biggest contribution of the paper.

Time burden: We expect you will need at most a day of active work to repeat all the experiments.

Prerequisites:
- Cloudlab cluster of 20 xl170 machines using `maestro_sosp24ae` Cloudlab profile (`https://www.cloudlab.us/p/faas-sched/maestro_sosp24ae`). 
- Chrome Cloudlab extension - install from https://github.com/eth-easl/cloudlab_extension

Order of experiments to run experiments:
- Start Cloudlab experiment

- Cold start sweep - containerd (instructions in `cold_start_sweep/dirigent`)
- Azure 500 - containerd (instructions in `azure_500/dirigent`)
- **Plot the data and verify** (run `run_plotting_scripts.sh`)
- **Reload the cluster through Cloudlab interface**


- Cold start sweep - Firecracker (instructions in `cold_start_sweep/dirigent`)
- Azure 500 - Firecracker (instructions in `azure_500/dirigent`)
- **Plot the data and verify** (run `run_plotting_scripts.sh`)
- **Reload the cluster through Cloudlab interface**


- Azure 500 - Knative/K8s
- Cold start sweep - Knative/K8s
- **Plot the data and verify** (run `run_plotting_scripts.sh`)

Note:
- Make sure to which addresses for of each node, because of the cluster utilization script. Node0 should run the loader, Node[1] and Node[1,2,3] run the Dirigent control plane(s) in non-HA and HA modes, respectively. Node[2] and Node[4,5,6] run the Dirigent data plane(s) in non-HA and HA modes, respectively. All the other nodes serve as worker nodes.
- All the plotting scripts are configured to work out of the box if you placed the experiment results in the correct folders.
- For Firecracker experiments, we noticed disk operation delays while creating Firecracker snapshots across different types of Cloudlab nodes. First 10 minutes of experiments on a new cluster may show a lot of timeouts. You should discard these measurements. The problems resolve after ~10 minutes on their own, assuming snapshots creation was triggered on each node (can be done with cold start sweep experiment).

Instructions to set up a Dirigent cluster:
- Make sure the cluster is in a reloaded state, i.e., that Dirigent is not running. 
- Clone Dirigent locally (`git clone https://github.com/eth-easl/dirigent.git`)
- Open Cloudlab experiment, open Cloudlab extension, and copy list of all addresses (RAW) using the extension. This puts the list of all nodes in your clipboard in format requested by the scripts below.
- Choose sandbox runtime (`containerd` or `firecracker`) by editing `WORKER_RUNTIME` in `./scripts/setup.cfg`
- Run locally `./scripts/remote_install.sh`. Arguments should be the copied list of addresses from the previous step. For example, `./scripts/remote_install.sh user@node1 user@node2 user@node3`. This script should be executed only once.
- Run locally `./scripts/remote_start_cluster.sh user@node1 user@node2 user@node3`. After this step Dirigent cluster will be operational. This script can be executed to restart Dirigent cluster in case you experience issues.

Instructions to set up Knative/K8s baseline cluster:
- TODO