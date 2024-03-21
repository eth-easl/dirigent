from common import *

rootPath = '.'
load = [500,1000,1500,2000]

labels = []
[labels.append(f'{x}  colds starts per second') for x in load]

plotClusteredStackedBarchart(getResult(load, rootPath),
                             title='',
                             clusterLabels=labels,
                             clusterLabelPosition=(-0.15, 1.1),
                             categoryLabelPosition=(-0.35, 0.65))

plt.title(f'Cold start latency breakdown sweep')
plt.xlabel('Percentile')
plt.xticks(rotation=0)
plt.ylabel('Latency [ms]')
plt.grid()
plt.tight_layout()

plt.savefig(f"{rootPath}/breakdown_sweep.png", dpi=160)
