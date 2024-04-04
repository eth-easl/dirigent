from common import *

rootPath = '/home/lcvetkovic/Desktop/replay/sosp/dirigent_1cp_1dp/rps_sweep_containerd/containerd'
load = [250, 1000, 1250]

#load = [500, 2000, 2500]

labels = []
[labels.append(f'{x} CS/s') for x in load]

rawData = getResult(load, rootPath)[0]
for idx in range(0, len(labels)):
    rawData[idx].to_csv(f'{rootPath}/statistics_load_{load[idx]}.csv')

plt.figure(figsize=(3, 6))
plotClusteredStackedBarchart(rawData,
                             title='',
                             clusterLabels=labels,
                             clusterLabelPosition=(0.15, 0.1),
                             categoryLabelPosition=(0.35, 0.65))

plt.title(f'Cold start latency\nbreakdown sweep')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown_sweep.png", dpi=160)
