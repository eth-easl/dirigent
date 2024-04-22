from common import *

df = getResult([1], '/home/lcvetkovic/Desktop/replay/sosp/dirigent_1cp_1dp/rps_sweep_containerd/containerd')[0]
df = df.rename(index={"p50": "Dirigent - containerd"})

fc = getResult([1], '/home/lcvetkovic/Desktop/replay/sosp/dirigent_1cp_1dp/rps_sweep_firecracker/firecracker')[0]
fc = fc.rename(index={"p50": "Dirigent - Firecracker"})

df = pd.concat([df, fc])

plt.figure()

df['Cluster manager'] = df['configure_monitoring'] + \
                        df['find_snapshot'] + \
                        df['other_worker_node'] + \
                        df['control_plane_other'] + \
                        df['serialization'] + \
                        df['persistence_layer'] + \
                        df['data_plane_propagation']

df['Other'] = df['get_metadata'] + \
              df['add_deployment'] + \
              df['load_balancing'] + \
              df['cc_throttling'] + \
              df['other'] + \
              df['image_fetch'] + \
              df['snapshot_creation']

df['Sandbox startup - function'] = df['sandbox_create'] + \
                                   df['sandbox_start']

df = df.drop(columns=[
    'get_metadata', 'add_deployment', 'load_balancing',
    'cc_throttling', 'serialization', 'persistence_layer',
    'other', 'image_fetch', 'readiness_probe',
    'data_plane_propagation', 'snapshot_creation', 'configure_monitoring',
    'find_snapshot', 'other_worker_node', 'control_plane_other',
    'sandbox_create', 'sandbox_start'
])

df = df.rename(columns={
    "network_setup": "Network allocation",
    "iptables": "iptables configuration",
    "readiness_probe": "Readiness check"
})

vhive = pd.DataFrame(data={
    'Cluster manager': [59.02],  # SUM(E287:H287)
    'Network allocation': [66],  # SUM(J287:L287)
    'Sandbox startup - function': [88.76],  # SUM(P287:Q287)
    'Sandbox startup - sidecar': [87.46],  # SUM(N287:O287)
    'Readiness check': [527], # R287
    'Other': [50.12 + 236.36],  # SUM(V287,T287,D287) + SUM(M287,R287)
}, index=['Knative-on-K8s'])

df['Cluster manager'] = 0
df = pd.concat([vhive, df])

df.plot(kind='bar', stacked=True, legend=True, figsize=(6, 2.5))

plt.xticks(rotation=0)
plt.ylabel('Median latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig("breakdown_sweep.pdf")
plt.savefig("breakdown_sweep.png")
