import pandas as pd

from clustered_plot import *

rootPath = './francois'
load = [800,801]


def processQuantile(d, percentile):
    p = d.reset_index()
    p = p.T
    p.columns = p.iloc[0]
    p = p.iloc[1:, :]
    p = p.rename(index={percentile: f"p{int(percentile * 100)}"})

    return p


result = []
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

    p50 = data.quantile(0.5)
    p95 = data.quantile(0.95)

    dataToPlot = processQuantile(p50, 0.5)
    dataToPlot = pd.concat([dataToPlot, processQuantile(p95, 0.95)])
    dataToPlot = dataToPlot / 1000  # μs -> ms

    result.append(dataToPlot)

plotClusteredStackedBarchart(result,
                             title='Sweep',
                             clusterLabels=[
                                 '1',
                                 '10',
                                 '25',
                                 '50',
                                 '75',
                                 '400 cold start',
                                 '800 cold start',
                             ],
                             clusterLabelPosition=(-0.335, 1.1),
                             categoryLabelPosition=(-0.25, 0.65))

plt.title(f'Cold start latency breakdown')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown.png")
