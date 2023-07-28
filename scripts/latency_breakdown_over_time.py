import pandas as pd

import matplotlib.pyplot as plt
import numpy as np


rootPath = './sweep'
load = [1,10,25]


def processQuantile(d, percentile):
    p = d.reset_index()
    p = p.T
    p.columns = p.iloc[0]
    p = p.iloc[1:, :]
    p = p.rename(index={percentile: f"p{int(percentile * 100)}"})

    return p


result = [1,10,25,50,75]
for l in load:
    cpTrace = pd.read_csv(f'{rootPath}/cold_start_trace_{l}.csv')
    proxyTrace = pd.read_csv(f'{rootPath}/proxy_trace_{l}.csv')



    data = pd.merge(proxyTrace, cpTrace, on=['container_id', 'service_name'], how='inner')
    data = data[data['success'] == True]  # keep only successful invocations
    data = data[data['cold_start'] > 0]  # keep only cold starts

    data = data.drop(columns=['time_x', 'time_y', 'success', 'service_name', 'container_id'])
    data = data.rename(columns={"other_x": "other_cp", "other_y": "other_worker_node"})

    data['control_plane'] = data['cold_start'] - \
                            (data['image_fetch'] + data['container_create'] + data['container_start'] +
                             data['cni'] + data['iptables'] + data['other_worker_node'])
    data = data.drop(columns=['cold_start'])

    data = data.to_numpy()
    data = np.sum(data, axis=1)

    y = data
    x = np.linspace(0, 60, data.shape[0])

    plt.plot(x, y, label=l)

plt.legend(loc="upper left")
plt.title(f'Sweep test')
plt.xlabel('Seconds')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()
plt.savefig(f"{rootPath}/breakdown_over_time.png")