import pandas as pd

from common import *

rootPath = './francois'
load = [50,100,200,400,800, 900, 1000,1600]


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

    p50 = data.quantile(0.5)
    p95 = data.quantile(0.95)

    dataToPlot = processQuantile(p50, 0.5)
    dataToPlot = pd.concat([dataToPlot, processQuantile(p95, 0.95)])
    dataToPlot = dataToPlot / 1000  # μs -> ms

    result.append(dataToPlot)

plotClusteredStackedBarchart(result,
                             title='',
                             clusterLabels=[
                                 '50 colds starts',
                                 '100 colds starts',
                                 '200 colds starts',
                                 '400 colds starts',
                                 '800 colds starts',
                                 '900 colds starts',
                                 '1000 colds starts',
                                 '1600 colds starts',
                             ],
                             clusterLabelPosition=(-0.335, 1.1),
                             categoryLabelPosition=(-0.25, 0.65))

plt.title(f'Cold start latency breakdown burst')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown_burst.png",dpi=160)