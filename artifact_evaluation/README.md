## Dirigent Artifact Evaluation Instructions

The following experiments aim to repeat results from Figures 7, 9, and 10, i.e., main results from sections 5.3 and 5.2.1, which constitute the biggest contribution of the paper. We chose not to describe Firecracker experiments as they are very complicated and time-consuming.

Time burden: We expect you will need at 3-5 hours to run the experiments we describe below.

Prerequisites:
- For the paper, we did all experiments on a 100-node Cloudlab `xl170` cluster. Because it is hard to get 100 nodes, we recommend using a cluster of at least 17  `xl170` Cloudlab machines running `maestro_sosp24ae` profile (`https://www.cloudlab.us/p/faas-sched/maestro_sosp24ae`), preferably 27. We provided sample results in the `artifact_evaluation` folder which we did on a 17-node cluster when we prepared artifacts.
- Chrome Cloudlab extension - install from https://github.com/eth-easl/cloudlab_extension

Order of experiments to run experiments:
- Start Cloudlab experiment

- Cold start sweep - containerd (instructions in `cold_start_sweep/dirigent`)
- Azure 500 - containerd (instructions in `azure_500/dirigent`)
- **Plot the data and verify** (run `run_plotting_scripts.sh`)
- **Reload the cluster through Cloudlab interface**


- Azure 500 - Knative/K8s (instructions in `azure_500/knative`)
- Cold start sweep - Knative/K8s (instructions in `cold_start_sweep/knative`)
- **Plot the data and verify** (run `run_plotting_scripts.sh`)

Notes:
- For simplicity and because we cannot guarantee artifact evaluators a huge Cloudlab cluster, we will be running all experiments in mode where Dirigent and Knative/K8s components are not replicated. We also ran experiment in an environment where components run in high-availability mode, but we did not notice significant performance differences. Our deployment scripts put load generator on `node0`, control plane on `node1`, data plane on `node2`, whereas all the other nodes serve as worker nodes.
- All the plotting scripts are configured to work out of the box provided you placed experiment results in correct folders.
- Traces for the experiments described here are stored on Git LFS. Make sure you pull these files before proceeding further.
- Default Cloudlab shell should be `bash`. You can configure when logged in here `https://www.cloudlab.us/myaccount.php`.

Instructions to set up a Dirigent cluster:
- Make sure the cluster is in a reloaded state, i.e., that neither Dirigent nor Knative is not running. 
- Clone Dirigent locally (`git clone https://github.com/eth-easl/dirigent.git`)
- Set sandbox runtime (`containerd`) by editing `WORKER_RUNTIME` in `./scripts/setup.cfg`
- Open Cloudlab experiment, open Cloudlab extension, and copy list of all addresses (RAW) using the extension. This puts the list of all nodes in your clipboard in format requested by the scripts below.
- Run locally `./scripts/remote_install.sh`. Arguments should be the copied list of addresses from the previous step. For example, `./scripts/remote_install.sh user@node0 user@node1 user@node2`. This script should be executed only once.
- Run locally `./scripts/remote_start_cluster.sh user@node0 user@node1 user@node2`. After this step Dirigent cluster should be operational. This script can be executed to restart Dirigent cluster in case you experience issues without reloading the Cloudlab cluster.

Instructions to set up Knative/K8s baseline cluster:
- Make sure the cluster is in a reloaded state, i.e., that neither Dirigent nor Knative is not running.
- Clone Invitro locally and checkout to `ha_k8s` branch (`git clone --branch=ha_k8s https://github.com/vhive-serverless/invitro`)
- Open Cloudlab experiment, open Cloudlab extension, and copy list of all addresses (RAW) using the extension. This puts the list of all nodes in your clipboard in format requested by the scripts below.
- Set up a Knative/K8s cluster by locally running `./scripts/setup/create_multinode.sh`. Arguments should be the copied list of addresses from the previous step. For example, `./scripts/setup/create_multinode.sh user@node0 user@node1 user@node2`. This script should be executed only once.
- After a couple of minutes, once the script has completed executing, the cluster should be running, and you can ssh into `node0`. Execute `kubectl get pods -A` and verify that installation has completed successfully by checking that all pods are in `Running` or `Completed` state.

Results expectation/interpretation:
- Since we cannot guarantee artifact evaluators access to a 100-node cluster over a 2-week artifact evaluation period, the performance on the smaller cluster will be slightly degraded.
  - For cold start sweep, the throughput we show in Figure 7 will be reduced, as worker nodes become the bottleneck. What you should verify is that the cold start throughput conforms to the following inequalities -- `Knative/K8s throughtput << Dirigent - containerd` and `Knative/K8s latency >> Dirigent - containerd latency`.
  - For Azure 500 trace experiments, the workload on Knative/K8s should be worse, and should suffer from a long tail. Per-invocation scheduling latency for Dirigent should be better almost all the time, and the average per-function scheduling latency of Dirigent should be by a couple of orders of magnitude better than with Knative/K8s.
  - You can see the results we got on a 17-node cluster in `sample_results` directory.