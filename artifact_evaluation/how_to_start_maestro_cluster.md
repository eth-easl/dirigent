Prerequisites:
- Cloudlab extension

Note:
- Make sure to which addresses for of each node, because of the cluster utilization script. Node0 should run the loader, Node[1] and Node[1,2,3] run the Maestro control plane(s) in non-HA and HA modes, respectively. Node[2] and Node[4,5,6] run the Maestro data plane(s) in non-HA and HA modes, respectively. All the other nodes serve as worker nodes.

Instructions:
- Make sure the cluster is in a reloaded state and that Maestro is not running. 
- Clone Maestro locally (git clone https://github.com/eth-easl/cluster_manager.git --branch v0.1.2)
- Edit remote_install.sh to checkout to branch v0.1.2
- Open Cloudlab experiment, open Cloudlab extension, and copy list of all addresses (RAW). This puts the list of all nodes in your clipboard in format requested by the scripts below.
- Run locally `./scripts/remote_install.sh`. Arguments should be the copied list of addresses from the previous step. For example, `./scripts/remote_install.sh user@node1 user@node2 user@node3`. This script should be executed only once.
- Run locally `./scripts/remote_start_cluster.sh user@node1 user@node2 user@node3`. After this step Maestro cluster will be operational. This script can be executed to restart Maestro cluster in case you experience issues.
