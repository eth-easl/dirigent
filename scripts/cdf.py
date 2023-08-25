import pandas as pd
import numpy as np
import matplotlib.pyplot as plt

rootPath = './francois'
load = [100,150,200,250,300,350,400,450]


percent = np.arange(1, 101, 1)

result = []
for l in load:
    cpTrace = pd.read_csv(f'{rootPath}/cold_start_trace_{l}.csv')
    proxyTrace = pd.read_csv(f'{rootPath}/proxy_trace_{l}.csv')

    data = pd.merge(proxyTrace, cpTrace, on=['container_id', 'service_name'], how='inner')
    data = data[data['success'] == True]  # keep only successful invocations
    data = data[data['cold_start'] > 0]  # keep only cold starts

    data = data.drop(columns=['time_x', 'time_y', 'success', 'service_name', 'container_id'])

    data['control_plane'] = data['cold_start'] - \
                            (data['image_fetch'] + data['container_create'] + data['container_start'] +
                             data['cni'] + data['iptables'] + data['db'] + data['other_worker_node'])
    data = data.drop(columns=['cold_start'])

    hist = []
    for per in percent:
        hist.append(np.sum(data.quantile(per / 100)))
    plt.plot(hist, percent,  label="{} colds starts per second".format(l))


plt.title(f'CDF - Sweep test')
plt.ylabel('Percentile')
plt.xlabel('Latency [ms]')
plt.legend()
plt.savefig(f"{rootPath}/cdf_sweep.png",dpi=160)